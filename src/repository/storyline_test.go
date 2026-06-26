package repository

import (
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestStorylineRepository_SearchNearest(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	repo := NewStorylineRepository(db)

	firstSeen := time.Date(2026, 6, 1, 0, 0, 0, 0, time.UTC)
	lastSeen := time.Date(2026, 6, 20, 0, 0, 0, 0, time.UTC)
	rows := sqlmock.NewRows([]string{"id", "channel_id", "title", "state", "category", "status", "importance", "first_seen", "last_seen", "similarity"}).
		AddRow(int64(42), int64(123), "Сюжет", "состояние", "политика", "active", 3, firstSeen, lastSeen, 0.87)

	mock.ExpectQuery("SELECT id, channel_id, title, state").
		WithArgs(int64(123), sqlmock.AnyArg(), 5).
		WillReturnRows(rows)

	results, err := repo.SearchNearest(123, []float32{0.1, 0.2, 0.3}, 5)
	require.NoError(t, err)
	require.Len(t, results, 1)
	assert.Equal(t, int64(42), results[0].Storyline.ID)
	assert.Equal(t, "Сюжет", results[0].Storyline.Title)
	assert.InDelta(t, 0.87, results[0].Similarity, 0.0001)

	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestStorylineRepository_GetStats(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	repo := NewStorylineRepository(db)

	lastSeen := time.Date(2026, 6, 18, 0, 0, 0, 0, time.UTC)
	rows := sqlmock.NewRows([]string{"days_seen", "median_count", "median_importance", "last_seen"}).
		AddRow(5, 3.0, 2.0, lastSeen)

	mock.ExpectQuery("FROM storyline_observations").
		WithArgs(int64(42), sqlmock.AnyArg(), sqlmock.AnyArg()).
		WillReturnRows(rows)

	stats, err := repo.GetStats(42, time.Date(2026, 6, 20, 0, 0, 0, 0, time.UTC), 14)
	require.NoError(t, err)
	assert.Equal(t, 5, stats.DaysSeen)
	assert.Equal(t, 3.0, stats.MedianCount)
	assert.Equal(t, 2.0, stats.MedianImportance)
	assert.Equal(t, lastSeen, stats.LastSeen)

	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestStorylineRepository_CreateStoryline(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	repo := NewStorylineRepository(db)

	day := time.Date(2026, 6, 20, 0, 0, 0, 0, time.UTC)
	s := &Storyline{
		ChannelID: 123, Title: "T", State: "S", Category: "экономика",
		Status: "active", Importance: 4, Embedding: []float32{0.1, 0.2}, FirstSeen: day, LastSeen: day,
	}

	mock.ExpectQuery("INSERT INTO storylines").
		WithArgs(int64(123), "T", "S", sqlmock.AnyArg(), "active", 4, sqlmock.AnyArg(), day, day).
		WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(int64(7)))

	id, err := repo.CreateStoryline(s)
	require.NoError(t, err)
	assert.Equal(t, int64(7), id)

	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestStorylineRepository_SaveObservationUpsert(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	repo := NewStorylineRepository(db)

	day := time.Date(2026, 6, 20, 0, 0, 0, 0, time.UTC)
	o := &Observation{
		StorylineID: 7, ChannelID: 123, ObsDate: day, MessageCount: 2, Importance: 3,
		ChangeType: "escalation", DeltaSummary: "новое", SourceMessageIDs: []int64{100, 101},
	}

	mock.ExpectExec("INSERT INTO storyline_observations").
		WithArgs(int64(7), int64(123), day, 2, 3, "escalation", sqlmock.AnyArg(), sqlmock.AnyArg()).
		WillReturnResult(sqlmock.NewResult(1, 1))

	err = repo.SaveObservation(o)
	require.NoError(t, err)

	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestStorylineRepository_DeleteObservationsForDate(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	repo := NewStorylineRepository(db)
	startOfDay := time.Date(2026, 6, 20, 0, 0, 0, 0, time.UTC)

	mock.ExpectExec("DELETE FROM storyline_observations").
		WithArgs(int64(123), startOfDay).
		WillReturnResult(sqlmock.NewResult(0, 3))

	err = repo.DeleteObservationsForDate(123, time.Date(2026, 6, 20, 15, 30, 0, 0, time.UTC))
	require.NoError(t, err)

	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestStorylineRepository_MarkDormant(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	repo := NewStorylineRepository(db)
	before := time.Date(2026, 6, 13, 0, 0, 0, 0, time.UTC)

	mock.ExpectExec("UPDATE storylines").
		WithArgs(int64(123), before).
		WillReturnResult(sqlmock.NewResult(0, 1))

	err = repo.MarkDormant(123, before)
	require.NoError(t, err)

	assert.NoError(t, mock.ExpectationsWereMet())
}
