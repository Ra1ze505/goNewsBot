package service

import (
	"context"
	"errors"
	"testing"
	"time"

	mock_repository "github.com/Ra1ze505/goNewsBot/src/mocks/repository"
	"github.com/Ra1ze505/goNewsBot/src/repository"
	"github.com/stretchr/testify/assert"
	"go.uber.org/mock/gomock"
)

func newTestSummaryService(summaryRepo *mock_repository.MockSummaryRepositoryInterface, storylineRepo *mock_repository.MockStorylineRepositoryInterface, mlRepo *mock_repository.MockMLRepositoryInterface) *SummaryService {
	processor := NewStorylineProcessor(summaryRepo, storylineRepo, mlRepo)
	messagesFetched := make(chan struct{})
	forceRegenerateChannel := make(chan struct{})
	return NewSummaryService(summaryRepo, storylineRepo, processor, messagesFetched, forceRegenerateChannel)
}

func TestSummaryService_ProcessChannelSummaries(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	summaryRepo := mock_repository.NewMockSummaryRepositoryInterface(ctrl)
	storylineRepo := mock_repository.NewMockStorylineRepositoryInterface(ctrl)
	mlRepo := mock_repository.NewMockMLRepositoryInterface(ctrl)
	service := newTestSummaryService(summaryRepo, storylineRepo, mlRepo)

	tests := []struct {
		name          string
		peerID        int64
		setupMocks    func()
		expectedError error
	}{
		{
			name:   "Successfully process channel summaries (no topics)",
			peerID: 123,
			setupMocks: func() {
				// Дайджест должен собираться за последний завершившийся UTC-день,
				// а не за текущий (в его начале сообщений ещё почти нет).
				isYesterdayUTC := gomock.Cond(func(x time.Time) bool {
					return x.Format("2006-01-02") == time.Now().UTC().AddDate(0, 0, -1).Format("2006-01-02")
				})
				summaryRepo.EXPECT().HasSummaryToday(int64(123)).Return(false, nil)
				storylineRepo.EXPECT().DeleteObservationsForDate(int64(123), isYesterdayUTC).Return(nil)
				summaryRepo.EXPECT().GetMessagesForDateWithIDs(int64(123), isYesterdayUTC).
					Return([]repository.MessageInput{{MessageID: 1, Text: "message1"}}, nil)
				mlRepo.EXPECT().ExtractTopics(gomock.Any()).Return(nil, nil)
				mlRepo.EXPECT().RenderDigest(gomock.Any()).Return("digest", nil)
				summaryRepo.EXPECT().SaveSummary(gomock.Any()).Return(nil)
			},
			expectedError: nil,
		},
		{
			name:   "Summary already exists for today",
			peerID: 123,
			setupMocks: func() {
				summaryRepo.EXPECT().HasSummaryToday(int64(123)).Return(true, nil)
			},
			expectedError: nil,
		},
		{
			name:   "Error checking summary existence",
			peerID: 123,
			setupMocks: func() {
				summaryRepo.EXPECT().HasSummaryToday(int64(123)).Return(false, errors.New("database error"))
			},
			expectedError: errors.New("failed to check summary existence for channel 123: database error"),
		},
		{
			name:   "No messages found",
			peerID: 123,
			setupMocks: func() {
				summaryRepo.EXPECT().HasSummaryToday(int64(123)).Return(false, nil)
				storylineRepo.EXPECT().DeleteObservationsForDate(int64(123), gomock.Any()).Return(nil)
				summaryRepo.EXPECT().GetMessagesForDateWithIDs(int64(123), gomock.Any()).Return(nil, nil)
			},
			expectedError: nil,
		},
		{
			name:   "Error getting messages",
			peerID: 123,
			setupMocks: func() {
				summaryRepo.EXPECT().HasSummaryToday(int64(123)).Return(false, nil)
				storylineRepo.EXPECT().DeleteObservationsForDate(int64(123), gomock.Any()).Return(nil)
				summaryRepo.EXPECT().GetMessagesForDateWithIDs(int64(123), gomock.Any()).Return(nil, errors.New("database error"))
			},
			expectedError: errors.New("failed to get messages for channel 123: database error"),
		},
		{
			name:   "Error saving summary",
			peerID: 123,
			setupMocks: func() {
				summaryRepo.EXPECT().HasSummaryToday(int64(123)).Return(false, nil)
				storylineRepo.EXPECT().DeleteObservationsForDate(int64(123), gomock.Any()).Return(nil)
				summaryRepo.EXPECT().GetMessagesForDateWithIDs(int64(123), gomock.Any()).
					Return([]repository.MessageInput{{MessageID: 1, Text: "message1"}}, nil)
				mlRepo.EXPECT().ExtractTopics(gomock.Any()).Return(nil, nil)
				mlRepo.EXPECT().RenderDigest(gomock.Any()).Return("digest", nil)
				summaryRepo.EXPECT().SaveSummary(gomock.Any()).Return(errors.New("database error"))
			},
			expectedError: errors.New("failed to save summary for channel 123: database error"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.setupMocks()
			err := service.ProcessChannelSummaries(tt.peerID)
			if tt.expectedError != nil {
				assert.Error(t, err)
				assert.Equal(t, tt.expectedError.Error(), err.Error())
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestSummaryService_StartSummaryFetcher(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	summaryRepo := mock_repository.NewMockSummaryRepositoryInterface(ctrl)
	storylineRepo := mock_repository.NewMockStorylineRepositoryInterface(ctrl)
	mlRepo := mock_repository.NewMockMLRepositoryInterface(ctrl)
	processor := NewStorylineProcessor(summaryRepo, storylineRepo, mlRepo)
	messagesFetched := make(chan struct{})
	forceRegenerateChannel := make(chan struct{})
	service := NewSummaryService(summaryRepo, storylineRepo, processor, messagesFetched, forceRegenerateChannel)

	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	summaryRepo.EXPECT().HasSummaryToday(gomock.Any()).Return(false, nil).AnyTimes()
	storylineRepo.EXPECT().DeleteObservationsForDate(gomock.Any(), gomock.Any()).Return(nil).AnyTimes()
	summaryRepo.EXPECT().GetMessagesForDateWithIDs(gomock.Any(), gomock.Any()).
		Return([]repository.MessageInput{{MessageID: 1, Text: "message1"}}, nil).AnyTimes()
	mlRepo.EXPECT().ExtractTopics(gomock.Any()).Return(nil, nil).AnyTimes()
	mlRepo.EXPECT().RenderDigest(gomock.Any()).Return("summary", nil).AnyTimes()
	summaryRepo.EXPECT().SaveSummary(gomock.Any()).Return(nil).AnyTimes()

	go service.StartSummaryFetcher(ctx)

	go func() {
		messagesFetched <- struct{}{}
	}()

	<-ctx.Done()
}
