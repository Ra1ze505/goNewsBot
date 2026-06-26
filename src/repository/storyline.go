package repository

//go:generate mockgen -source=storyline.go -destination=../mocks/repository/storyline_mock.go -package=mock_repository

import (
	"database/sql"
	"time"

	"github.com/lib/pq"
	"github.com/pgvector/pgvector-go"
)

type Storyline struct {
	ID         int64
	ChannelID  int64
	Title      string
	State      string
	Category   string
	Status     string // active | dormant | closed
	Importance int
	Embedding  []float32 // 256
	FirstSeen  time.Time
	LastSeen   time.Time
}

type Observation struct {
	StorylineID      int64
	ChannelID        int64
	ObsDate          time.Time
	MessageCount     int
	Importance       int
	ChangeType       string
	DeltaSummary     string
	SourceMessageIDs []int64
}

// StorylineStats - агрегаты по observations со obs_date < date (окно BaselineWindowDays).
type StorylineStats struct {
	DaysSeen         int
	MedianCount      float64
	MedianImportance float64
	LastSeen         time.Time
}

type ScoredStoryline struct {
	Storyline  Storyline
	Similarity float64
}

type StorylineRepositoryInterface interface {
	// матчинг
	SearchNearest(channelID int64, query []float32, k int) ([]ScoredStoryline, error) // active only
	GetActive(channelID int64) ([]Storyline, error)

	// статистика для классификации (строго obs_date < before)
	GetStats(storylineID int64, before time.Time, windowDays int) (StorylineStats, error)

	// запись состояния
	CreateStoryline(s *Storyline) (int64, error)
	UpdateStoryline(s *Storyline) error
	SaveObservation(o *Observation) error // upsert по (storyline_id, obs_date)

	// жизненный цикл
	MarkDormant(channelID int64, lastSeenBefore time.Time) error
	MarkClosed(channelID int64, lastSeenBefore time.Time) error

	// идемпотентность перегенерации/бэкфилла
	DeleteObservationsForDate(channelID int64, date time.Time) error
	ResetChannel(channelID int64) error
}

type StorylineRepository struct {
	db *sql.DB
}

func NewStorylineRepository(db *sql.DB) StorylineRepositoryInterface {
	return &StorylineRepository{db: db}
}

func (r *StorylineRepository) SearchNearest(channelID int64, query []float32, k int) ([]ScoredStoryline, error) {
	q := `
		SELECT id, channel_id, title, state, COALESCE(category, ''), status, importance, first_seen, last_seen,
			1 - (embedding <=> $2) AS similarity
		FROM storylines
		WHERE channel_id = $1 AND status = 'active' AND embedding IS NOT NULL
		ORDER BY embedding <=> $2
		LIMIT $3
	`
	rows, err := r.db.Query(q, channelID, pgvector.NewVector(query), k)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var results []ScoredStoryline
	for rows.Next() {
		var s Storyline
		var sim float64
		if err := rows.Scan(
			&s.ID, &s.ChannelID, &s.Title, &s.State, &s.Category, &s.Status, &s.Importance,
			&s.FirstSeen, &s.LastSeen, &sim,
		); err != nil {
			return nil, err
		}
		results = append(results, ScoredStoryline{Storyline: s, Similarity: sim})
	}
	return results, rows.Err()
}

func (r *StorylineRepository) GetActive(channelID int64) ([]Storyline, error) {
	q := `
		SELECT id, channel_id, title, state, COALESCE(category, ''), status, importance, first_seen, last_seen
		FROM storylines
		WHERE channel_id = $1 AND status = 'active'
		ORDER BY last_seen DESC
	`
	rows, err := r.db.Query(q, channelID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var results []Storyline
	for rows.Next() {
		var s Storyline
		if err := rows.Scan(
			&s.ID, &s.ChannelID, &s.Title, &s.State, &s.Category, &s.Status, &s.Importance,
			&s.FirstSeen, &s.LastSeen,
		); err != nil {
			return nil, err
		}
		results = append(results, s)
	}
	return results, rows.Err()
}

func (r *StorylineRepository) GetStats(storylineID int64, before time.Time, windowDays int) (StorylineStats, error) {
	windowStart := before.AddDate(0, 0, -windowDays)
	q := `
		SELECT
			COUNT(*) AS days_seen,
			COALESCE(percentile_cont(0.5) WITHIN GROUP (ORDER BY message_count), 0) AS median_count,
			COALESCE(percentile_cont(0.5) WITHIN GROUP (ORDER BY importance), 0) AS median_importance,
			MAX(obs_date) AS last_seen
		FROM storyline_observations
		WHERE storyline_id = $1 AND obs_date < $2 AND obs_date >= $3
	`
	var stats StorylineStats
	var lastSeen sql.NullTime
	err := r.db.QueryRow(q, storylineID, before, windowStart).Scan(
		&stats.DaysSeen, &stats.MedianCount, &stats.MedianImportance, &lastSeen,
	)
	if err != nil {
		return StorylineStats{}, err
	}
	if lastSeen.Valid {
		stats.LastSeen = lastSeen.Time
	}
	return stats, nil
}

func (r *StorylineRepository) CreateStoryline(s *Storyline) (int64, error) {
	q := `
		INSERT INTO storylines (channel_id, title, state, category, status, importance, embedding, first_seen, last_seen)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
		RETURNING id
	`
	var id int64
	err := r.db.QueryRow(q,
		s.ChannelID, s.Title, s.State, nullString(s.Category), statusOrActive(s.Status), s.Importance,
		pgvector.NewVector(s.Embedding), s.FirstSeen, s.LastSeen,
	).Scan(&id)
	if err != nil {
		return 0, err
	}
	return id, nil
}

func (r *StorylineRepository) UpdateStoryline(s *Storyline) error {
	q := `
		UPDATE storylines
		SET title = $2, state = $3, category = $4, status = $5, importance = $6,
			embedding = $7, last_seen = $8, updated_at = CURRENT_TIMESTAMP
		WHERE id = $1
	`
	_, err := r.db.Exec(q,
		s.ID, s.Title, s.State, nullString(s.Category), statusOrActive(s.Status), s.Importance,
		pgvector.NewVector(s.Embedding), s.LastSeen,
	)
	return err
}

func (r *StorylineRepository) SaveObservation(o *Observation) error {
	q := `
		INSERT INTO storyline_observations
			(storyline_id, channel_id, obs_date, message_count, importance, change_type, delta_summary, source_message_ids)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		ON CONFLICT (storyline_id, obs_date) DO UPDATE SET
			message_count = EXCLUDED.message_count,
			importance = EXCLUDED.importance,
			change_type = EXCLUDED.change_type,
			delta_summary = EXCLUDED.delta_summary,
			source_message_ids = EXCLUDED.source_message_ids
	`
	_, err := r.db.Exec(q,
		o.StorylineID, o.ChannelID, o.ObsDate, o.MessageCount, o.Importance,
		o.ChangeType, nullString(o.DeltaSummary), pq.Array(o.SourceMessageIDs),
	)
	return err
}

func (r *StorylineRepository) MarkDormant(channelID int64, lastSeenBefore time.Time) error {
	q := `
		UPDATE storylines
		SET status = 'dormant', updated_at = CURRENT_TIMESTAMP
		WHERE channel_id = $1 AND status = 'active' AND last_seen < $2
	`
	_, err := r.db.Exec(q, channelID, lastSeenBefore)
	return err
}

func (r *StorylineRepository) MarkClosed(channelID int64, lastSeenBefore time.Time) error {
	q := `
		UPDATE storylines
		SET status = 'closed', updated_at = CURRENT_TIMESTAMP
		WHERE channel_id = $1 AND status <> 'closed' AND last_seen < $2
	`
	_, err := r.db.Exec(q, channelID, lastSeenBefore)
	return err
}

func (r *StorylineRepository) DeleteObservationsForDate(channelID int64, date time.Time) error {
	startOfDay := time.Date(date.Year(), date.Month(), date.Day(), 0, 0, 0, 0, time.UTC)
	q := `DELETE FROM storyline_observations WHERE channel_id = $1 AND obs_date = $2`
	_, err := r.db.Exec(q, channelID, startOfDay)
	return err
}

func (r *StorylineRepository) ResetChannel(channelID int64) error {
	if _, err := r.db.Exec(`DELETE FROM storyline_observations WHERE channel_id = $1`, channelID); err != nil {
		return err
	}
	_, err := r.db.Exec(`DELETE FROM storylines WHERE channel_id = $1`, channelID)
	return err
}

func nullString(s string) sql.NullString {
	if s == "" {
		return sql.NullString{}
	}
	return sql.NullString{String: s, Valid: true}
}

func statusOrActive(s string) string {
	if s == "" {
		return "active"
	}
	return s
}
