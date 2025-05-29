package repository

//go:generate mockgen -source=summary.go -destination=../mocks/repository/summary_mock.go -package=mock_repository

import (
	"database/sql"
	"fmt"
	"time"
)

type Summary struct {
	ID        int64
	ChannelID int64
	Summary   string
	CreatedAt time.Time
}

func (s *Summary) GetFormattedSummary() string {
	return fmt.Sprintf("Последние новости:\n%s\n\nСуммаризация от %s UTC", s.Summary, s.CreatedAt.Format("2006-01-02 15:04:05"))
}

type SummaryRepositoryInterface interface {
	SaveSummary(summary *Summary) error
	HasSummaryToday(channelID int64) (bool, error)
	GetMessagesForLastDay(channelID int64) ([]string, error)
	GetLatestSummary(channelID int64) (*Summary, error)
}

type SummaryRepository struct {
	db *sql.DB
}

func NewSummaryRepository(db *sql.DB) SummaryRepositoryInterface {
	return &SummaryRepository{db: db}
}

func (r *SummaryRepository) SaveSummary(summary *Summary) error {
	query := `
		INSERT INTO summaries (channel_id, summary, created_at)
		VALUES ($1, $2, $3)
	`
	_, err := r.db.Exec(query,
		summary.ChannelID,
		summary.Summary,
		summary.CreatedAt,
	)
	return err
}

func (r *SummaryRepository) HasSummaryToday(channelID int64) (bool, error) {
	var count int
	query := `
		SELECT COUNT(*) 
		FROM summaries 
		WHERE channel_id = $1 
		AND DATE(created_at) = CURRENT_DATE
	`
	err := r.db.QueryRow(query, channelID).Scan(&count)
	if err != nil {
		return false, err
	}
	return count > 0, nil
}

func (r *SummaryRepository) GetMessagesForLastDay(channelID int64) ([]string, error) {
	query := `
		SELECT message_text 
		FROM messages 
		WHERE channel_id = $1 
		AND message_date >= NOW() - INTERVAL '1 day'
		ORDER BY message_date ASC
	`
	rows, err := r.db.Query(query, channelID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var messages []string
	for rows.Next() {
		var text string
		if err := rows.Scan(&text); err != nil {
			return nil, err
		}
		messages = append(messages, text)
	}
	return messages, rows.Err()
}

func (r *SummaryRepository) GetLatestSummary(channelID int64) (*Summary, error) {
	query := `
		SELECT id, channel_id, summary, created_at
		FROM summaries
		WHERE channel_id = $1
		ORDER BY created_at DESC
		LIMIT 1
	`
	summary := &Summary{}
	err := r.db.QueryRow(query, channelID).Scan(
		&summary.ID,
		&summary.ChannelID,
		&summary.Summary,
		&summary.CreatedAt,
	)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return summary, nil
}
