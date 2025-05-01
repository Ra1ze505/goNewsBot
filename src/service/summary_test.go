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
	service := NewSummaryService(summaryRepo, mlRepo)

	tests := []struct {
		name            string
		channelUsername string
		setupMocks      func()
		expectedError   error
	}{
		{
			name:            "Successfully process channel summaries",
			channelUsername: "test_channel",
			setupMocks: func() {
				summaryRepo.EXPECT().GetChannelID("test_channel").Return(int64(123), nil)
				summaryRepo.EXPECT().HasSummaryToday(int64(123)).Return(false, nil)
				summaryRepo.EXPECT().GetMessagesForLastDay(int64(123)).Return([]string{"message1", "message2"}, nil)
				mlRepo.EXPECT().SummarizeMessages([]string{"message1", "message2"}).Return("summary", nil)
				summaryRepo.EXPECT().SaveSummary(gomock.Any()).Return(nil)
			},
			expectedError: nil,
		},
		{
			name:            "Channel not found",
			channelUsername: "non_existent_channel",
			setupMocks: func() {
				summaryRepo.EXPECT().GetChannelID("non_existent_channel").Return(int64(0), nil)
			},
			expectedError: nil,
		},
		{
			name:            "Summary already exists for today",
			channelUsername: "test_channel",
			setupMocks: func() {
				summaryRepo.EXPECT().GetChannelID("test_channel").Return(int64(123), nil)
				summaryRepo.EXPECT().HasSummaryToday(int64(123)).Return(true, nil)
			},
			expectedError: nil,
		},
		{
			name:            "Error getting channel ID",
			channelUsername: "test_channel",
			setupMocks: func() {
				summaryRepo.EXPECT().GetChannelID("test_channel").Return(int64(0), errors.New("database error"))
			},
			expectedError: errors.New("database error"),
		},
		{
			name:            "Error checking summary existence",
			channelUsername: "test_channel",
			setupMocks: func() {
				summaryRepo.EXPECT().GetChannelID("test_channel").Return(int64(123), nil)
				summaryRepo.EXPECT().HasSummaryToday(int64(123)).Return(false, errors.New("database error"))
			},
			expectedError: errors.New("database error"),
		},
		{
			name:            "Error getting messages",
			channelUsername: "test_channel",
			setupMocks: func() {
				summaryRepo.EXPECT().GetChannelID("test_channel").Return(int64(123), nil)
				summaryRepo.EXPECT().HasSummaryToday(int64(123)).Return(false, nil)
				summaryRepo.EXPECT().GetMessagesForLastDay(int64(123)).Return(nil, errors.New("database error"))
			},
			expectedError: errors.New("database error"),
		},
		{
			name:            "Error summarizing messages",
			channelUsername: "test_channel",
			setupMocks: func() {
				summaryRepo.EXPECT().GetChannelID("test_channel").Return(int64(123), nil)
				summaryRepo.EXPECT().HasSummaryToday(int64(123)).Return(false, nil)
				summaryRepo.EXPECT().GetMessagesForLastDay(int64(123)).Return([]string{"message1", "message2"}, nil)
				mlRepo.EXPECT().SummarizeMessages([]string{"message1", "message2"}).Return("", errors.New("ml service error"))
			},
			expectedError: errors.New("ml service error"),
		},
		{
			name:            "Error saving summary",
			channelUsername: "test_channel",
			setupMocks: func() {
				summaryRepo.EXPECT().GetChannelID("test_channel").Return(int64(123), nil)
				summaryRepo.EXPECT().HasSummaryToday(int64(123)).Return(false, nil)
				summaryRepo.EXPECT().GetMessagesForLastDay(int64(123)).Return([]string{"message1", "message2"}, nil)
				mlRepo.EXPECT().SummarizeMessages([]string{"message1", "message2"}).Return("summary", nil)
				summaryRepo.EXPECT().SaveSummary(gomock.Any()).Return(errors.New("database error"))
			},
			expectedError: errors.New("database error"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.setupMocks()
			err := service.ProcessChannelSummaries(tt.channelUsername)
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
	service := NewSummaryService(summaryRepo, mlRepo)

	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	// Mock the processAllChannels call that happens on startup
	summaryRepo.EXPECT().GetChannelID(gomock.Any()).Return(int64(123), nil)
	summaryRepo.EXPECT().HasSummaryToday(int64(123)).Return(false, nil)
	summaryRepo.EXPECT().GetMessagesForLastDay(int64(123)).Return([]string{"message1"}, nil)
	mlRepo.EXPECT().SummarizeMessages([]string{"message1"}).Return("summary", nil)
	summaryRepo.EXPECT().SaveSummary(gomock.Any()).Return(nil)

	// Start the fetcher in a goroutine
	go service.StartSummaryFetcher(ctx)

	// Wait for the context to be done
	<-ctx.Done()
}
