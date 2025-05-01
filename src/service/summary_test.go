package service

import (
	"context"
	"errors"
	"testing"
	"time"

	mock_repository "github.com/Ra1ze505/goNewsBot/src/mocks/repository"
	"github.com/stretchr/testify/assert"
	"go.uber.org/mock/gomock"
)

func TestSummaryService_ProcessChannelSummaries(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	summaryRepo := mock_repository.NewMockSummaryRepositoryInterface(ctrl)
	mlRepo := mock_repository.NewMockMLRepositoryInterface(ctrl)
	messagesFetched := make(chan struct{})
	service := NewSummaryService(summaryRepo, mlRepo, messagesFetched)

	tests := []struct {
		name          string
		peerID        int64
		setupMocks    func()
		expectedError error
	}{
		{
			name:   "Successfully process channel summaries",
			peerID: 123,
			setupMocks: func() {
				summaryRepo.EXPECT().HasSummaryToday(int64(123)).Return(false, nil)
				summaryRepo.EXPECT().GetMessagesForLastDay(int64(123)).Return([]string{"message1", "message2"}, nil)
				mlRepo.EXPECT().SummarizeMessages([]string{"message1", "message2"}).Return("summary", nil)
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
			name:   "Error getting messages",
			peerID: 123,
			setupMocks: func() {
				summaryRepo.EXPECT().HasSummaryToday(int64(123)).Return(false, nil)
				summaryRepo.EXPECT().GetMessagesForLastDay(int64(123)).Return(nil, errors.New("database error"))
			},
			expectedError: errors.New("failed to get messages for channel 123: database error"),
		},
		{
			name:   "Error summarizing messages",
			peerID: 123,
			setupMocks: func() {
				summaryRepo.EXPECT().HasSummaryToday(int64(123)).Return(false, nil)
				summaryRepo.EXPECT().GetMessagesForLastDay(int64(123)).Return([]string{"message1", "message2"}, nil)
				mlRepo.EXPECT().SummarizeMessages([]string{"message1", "message2"}).Return("", errors.New("ml service error"))
			},
			expectedError: errors.New("failed to generate summary for channel 123: ml service error"),
		},
		{
			name:   "Error saving summary",
			peerID: 123,
			setupMocks: func() {
				summaryRepo.EXPECT().HasSummaryToday(int64(123)).Return(false, nil)
				summaryRepo.EXPECT().GetMessagesForLastDay(int64(123)).Return([]string{"message1", "message2"}, nil)
				mlRepo.EXPECT().SummarizeMessages([]string{"message1", "message2"}).Return("summary", nil)
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
	mlRepo := mock_repository.NewMockMLRepositoryInterface(ctrl)
	messagesFetched := make(chan struct{})
	service := NewSummaryService(summaryRepo, mlRepo, messagesFetched)

	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	// Set up expectations for each channel operation
	summaryRepo.EXPECT().HasSummaryToday(gomock.Any()).Return(false, nil).AnyTimes()
	summaryRepo.EXPECT().GetMessagesForLastDay(gomock.Any()).Return([]string{"message1"}, nil).AnyTimes()
	mlRepo.EXPECT().SummarizeMessages([]string{"message1"}).Return("summary", nil).AnyTimes()
	summaryRepo.EXPECT().SaveSummary(gomock.Any()).Return(nil).AnyTimes()

	go service.StartSummaryFetcher(ctx)

	// Send a message fetched signal
	go func() {
		messagesFetched <- struct{}{}
	}()

	<-ctx.Done()
}
