package service

import (
	"testing"
	"time"

	mock_repository "github.com/Ra1ze505/goNewsBot/src/mocks/repository"
	"github.com/Ra1ze505/goNewsBot/src/repository"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

func TestProcessDay_NewStoryline(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	summaryRepo := mock_repository.NewMockSummaryRepositoryInterface(ctrl)
	storylineRepo := mock_repository.NewMockStorylineRepositoryInterface(ctrl)
	mlRepo := mock_repository.NewMockMLRepositoryInterface(ctrl)
	processor := NewStorylineProcessor(summaryRepo, storylineRepo, mlRepo)

	day := time.Date(2026, 6, 20, 12, 0, 0, 0, time.UTC)
	msgs := []repository.MessageInput{{MessageID: 100, Text: "Произошло крупное событие"}}

	embedding := make([]float32, 256)

	mlRepo.EXPECT().ExtractTopics(msgs).Return([]repository.CandidateTopic{
		{Title: "Событие", Summary: "Описание", Category: "происшествия", Importance: 4, SourceMessageNumbers: []int{1}},
	}, nil)
	mlRepo.EXPECT().EmbedQueries([]string{"Событие\nОписание"}).Return([][]float32{embedding}, nil)
	storylineRepo.EXPECT().SearchNearest(int64(123), embedding, 5).Return(nil, nil)
	mlRepo.EXPECT().WriteDelta(gomock.Any()).Return("новое состояние", "что нового", nil)
	mlRepo.EXPECT().EmbedDocuments([]string{"Событие\nновое состояние"}).Return([][]float32{embedding}, nil)

	storylineRepo.EXPECT().CreateStoryline(gomock.Any()).DoAndReturn(func(s *repository.Storyline) (int64, error) {
		assert.Equal(t, "Событие", s.Title)
		assert.Equal(t, "новое состояние", s.State)
		assert.Equal(t, int64(123), s.ChannelID)
		assert.Equal(t, day.Truncate(24*time.Hour), s.FirstSeen)
		return int64(7), nil
	})
	storylineRepo.EXPECT().SaveObservation(gomock.Any()).DoAndReturn(func(o *repository.Observation) error {
		assert.Equal(t, int64(7), o.StorylineID)
		assert.Equal(t, "new", o.ChangeType)
		assert.Equal(t, []int64{100}, o.SourceMessageIDs)
		assert.Equal(t, 1, o.MessageCount)
		return nil
	})
	storylineRepo.EXPECT().MarkDormant(int64(123), gomock.Any()).Return(nil)
	storylineRepo.EXPECT().MarkClosed(int64(123), gomock.Any()).Return(nil)
	mlRepo.EXPECT().RenderDigest(gomock.Any()).DoAndReturn(func(groups repository.DigestGroups) (string, error) {
		require.Len(t, groups.New, 1)
		assert.Equal(t, "Событие", groups.New[0].Title)
		return "итоговый дайджест", nil
	})

	digest, err := processor.ProcessDay(123, day, msgs)
	require.NoError(t, err)
	assert.Equal(t, "итоговый дайджест", digest)
}

func TestProcessDay_MatchExistingHighSimilarity(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	summaryRepo := mock_repository.NewMockSummaryRepositoryInterface(ctrl)
	storylineRepo := mock_repository.NewMockStorylineRepositoryInterface(ctrl)
	mlRepo := mock_repository.NewMockMLRepositoryInterface(ctrl)
	processor := NewStorylineProcessor(summaryRepo, storylineRepo, mlRepo)

	day := time.Date(2026, 6, 20, 0, 0, 0, 0, time.UTC)
	msgs := []repository.MessageInput{{MessageID: 5, Text: "Развитие сюжета"}}
	embedding := make([]float32, 256)

	existing := repository.Storyline{ID: 42, ChannelID: 123, Title: "Существующий", State: "старое", Category: "политика", LastSeen: day.AddDate(0, 0, -1)}

	mlRepo.EXPECT().ExtractTopics(msgs).Return([]repository.CandidateTopic{
		{Title: "Развитие", Summary: "Детали", Category: "политика", Importance: 3, SourceMessageNumbers: []int{1}},
	}, nil)
	mlRepo.EXPECT().EmbedQueries(gomock.Any()).Return([][]float32{embedding}, nil)
	storylineRepo.EXPECT().SearchNearest(int64(123), embedding, 5).Return([]repository.ScoredStoryline{
		{Storyline: existing, Similarity: 0.92},
	}, nil)
	storylineRepo.EXPECT().GetStats(int64(42), gomock.Any(), 14).Return(repository.StorylineStats{
		DaysSeen: 3, MedianCount: 2, MedianImportance: 3,
	}, nil)
	mlRepo.EXPECT().WriteDelta(gomock.Any()).Return("обновлённое состояние", "сегодня новое", nil)
	mlRepo.EXPECT().EmbedDocuments(gomock.Any()).Return([][]float32{embedding}, nil)
	storylineRepo.EXPECT().UpdateStoryline(gomock.Any()).DoAndReturn(func(s *repository.Storyline) error {
		assert.Equal(t, int64(42), s.ID)
		assert.Equal(t, "Существующий", s.Title)
		assert.Equal(t, "обновлённое состояние", s.State)
		return nil
	})
	storylineRepo.EXPECT().SaveObservation(gomock.Any()).Return(nil)
	storylineRepo.EXPECT().MarkDormant(int64(123), gomock.Any()).Return(nil)
	storylineRepo.EXPECT().MarkClosed(int64(123), gomock.Any()).Return(nil)
	mlRepo.EXPECT().RenderDigest(gomock.Any()).Return("дайджест", nil)

	digest, err := processor.ProcessDay(123, day, msgs)
	require.NoError(t, err)
	assert.Equal(t, "дайджест", digest)
}

func TestClassifyChangeType(t *testing.T) {
	// эскалация по объёму
	assert.Equal(t, "escalation", classifyChangeType(repository.StorylineStats{MedianCount: 2, MedianImportance: 2, DaysSeen: 3}, 5, 2))
	// эскалация по важности
	assert.Equal(t, "escalation", classifyChangeType(repository.StorylineStats{MedianCount: 2, MedianImportance: 1, DaysSeen: 3}, 2, 4))
	// шум: частый, низкая важность, объём около baseline
	assert.Equal(t, "recurring_noise", classifyChangeType(repository.StorylineStats{MedianCount: 3, MedianImportance: 1, DaysSeen: 10}, 3, 1))
	// деэскалация
	assert.Equal(t, "deescalation", classifyChangeType(repository.StorylineStats{MedianCount: 10, MedianImportance: 3, DaysSeen: 5}, 1, 3))
	// иначе ongoing
	assert.Equal(t, "ongoing", classifyChangeType(repository.StorylineStats{MedianCount: 3, MedianImportance: 3, DaysSeen: 2}, 4, 3))
}

func TestBuildDigestGroups(t *testing.T) {
	entries := []digestEntry{
		{title: "A", changeType: "new", deltaSummary: "x"},
		{title: "B", changeType: "escalation", deltaSummary: "y"},
		{title: "C", changeType: "ongoing", deltaSummary: "z"},
		{title: "D", changeType: "ongoing", deltaSummary: ""},
		{title: "E", changeType: "recurring_noise", category: "происшествия"},
		{title: "F", changeType: "recurring_noise", category: "происшествия"},
	}
	groups := buildDigestGroups(entries)
	assert.Len(t, groups.New, 1)
	assert.Len(t, groups.Escalation, 1)
	assert.Len(t, groups.Ongoing, 1) // D отброшен из-за пустой дельты
	assert.Equal(t, []string{"происшествия"}, groups.RecurringNoise)
}
