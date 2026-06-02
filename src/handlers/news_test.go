package handlers

import (
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/Ra1ze505/goNewsBot/src/keyboard"
	mock_repository "github.com/Ra1ze505/goNewsBot/src/mocks/repository"
	mock_telebot "github.com/Ra1ze505/goNewsBot/src/mocks/telebot"
	"github.com/Ra1ze505/goNewsBot/src/repository"
	"github.com/Ra1ze505/goNewsBot/src/telegramutil"
	"go.uber.org/mock/gomock"
	tele "gopkg.in/telebot.v4"
)

func TestNewsHandler_Handle(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockSummaryRepo := mock_repository.NewMockSummaryRepositoryInterface(ctrl)
	mockContext := mock_telebot.NewMockContext(ctrl)

	testUserID := 1
	testUser := &repository.User{
		ID:                 &testUserID,
		PreferredChannelID: 123,
	}
	testSummary := &repository.Summary{
		ID:        1,
		ChannelID: 123,
		Summary:   "Test summary content",
		CreatedAt: time.Now(),
	}

	tests := []struct {
		name        string
		summary     *repository.Summary
		summaryErr  error
		expectedErr error
		setupMocks  func()
	}{
		{
			name:        "Success case",
			summary:     testSummary,
			summaryErr:  nil,
			expectedErr: nil,
			setupMocks: func() {
				mockContext.EXPECT().Get("user").Return(testUser)
				mockSummaryRepo.EXPECT().GetLatestSummary(testUser.PreferredChannelID).Return(testSummary, nil)
				mockContext.EXPECT().Send(
					"Последние новости:\nTest summary content\n\nСуммаризация от "+testSummary.CreatedAt.Format("2006-01-02 15:04:05")+" UTC",
					keyboard.GetStartKeyboard(),
					&tele.SendOptions{ParseMode: tele.ModeMarkdown},
				).Return(nil)
			},
		},
		{
			name:        "No news available",
			summary:     nil,
			summaryErr:  nil,
			expectedErr: nil,
			setupMocks: func() {
				mockContext.EXPECT().Get("user").Return(testUser)
				mockSummaryRepo.EXPECT().GetLatestSummary(testUser.PreferredChannelID).Return(nil, nil)
				mockContext.EXPECT().Send("Новостей пока нет. Проверьте позже.", keyboard.GetStartKeyboard()).Return(nil)
			},
		},
		{
			name:        "Database error",
			summary:     nil,
			summaryErr:  errors.New("database error"),
			expectedErr: nil,
			setupMocks: func() {
				mockContext.EXPECT().Get("user").Return(testUser)
				mockSummaryRepo.EXPECT().GetLatestSummary(testUser.PreferredChannelID).Return(nil, errors.New("database error"))
				mockContext.EXPECT().Send("Произошла ошибка при получении новостей. Попробуйте позже.", keyboard.GetStartKeyboard()).Return(nil)
			},
		},
		{
			name:        "User not found in context",
			summary:     nil,
			summaryErr:  nil,
			expectedErr: errors.New("user not found in context"),
			setupMocks: func() {
				mockContext.EXPECT().Get("user").Return(nil)
			},
		},
		{
			name: "Summary too long",
			summary: &repository.Summary{
				ID:        1,
				ChannelID: 123,
				Summary:   strings.Repeat("a", 4097),
				CreatedAt: time.Date(2026, 6, 3, 10, 0, 0, 0, time.UTC),
			},
			summaryErr:  nil,
			expectedErr: nil,
			setupMocks: func() {
				longSummary := &repository.Summary{
					ID:        1,
					ChannelID: 123,
					Summary:   strings.Repeat("a", 4097),
					CreatedAt: time.Date(2026, 6, 3, 10, 0, 0, 0, time.UTC),
				}
				fullMessage := longSummary.GetFormattedSummary()
				parts := telegramutil.SplitMessage(fullMessage)

				mockContext.EXPECT().Get("user").Return(testUser)
				mockSummaryRepo.EXPECT().GetLatestSummary(testUser.PreferredChannelID).Return(longSummary, nil)

				for i, part := range parts {
					withKeyboard := i == len(parts)-1
					if withKeyboard {
						mockContext.EXPECT().Send(
							part,
							keyboard.GetStartKeyboard(),
							&tele.SendOptions{ParseMode: tele.ModeMarkdown},
						).Return(nil)
					} else {
						mockContext.EXPECT().Send(
							part,
							&tele.SendOptions{ParseMode: tele.ModeMarkdown},
						).Return(nil)
					}
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.setupMocks()

			handler := NewNewsHandler(mockSummaryRepo)

			err := handler.Handle(mockContext)

			if err != nil && tt.expectedErr != nil {
				if err.Error() != tt.expectedErr.Error() {
					t.Errorf("Handle() error = %v, expectedErr %v", err, tt.expectedErr)
				}
			} else if err != tt.expectedErr {
				t.Errorf("Handle() error = %v, expectedErr %v", err, tt.expectedErr)
			}
		})
	}
}

func TestNewsHandler_HandleLongSummarySplit(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockSummaryRepo := mock_repository.NewMockSummaryRepositoryInterface(ctrl)
	mockContext := mock_telebot.NewMockContext(ctrl)

	testUser := &repository.User{
		ID:                 &[]int{1}[0],
		PreferredChannelID: 123,
	}
	longSummary := &repository.Summary{
		ID:        1,
		ChannelID: 123,
		Summary:   strings.Repeat("a", 4097),
		CreatedAt: time.Date(2026, 6, 3, 10, 0, 0, 0, time.UTC),
	}
	fullMessage := longSummary.GetFormattedSummary()
	expectedParts := telegramutil.SplitMessage(fullMessage)

	mockContext.EXPECT().Get("user").Return(testUser)
	mockSummaryRepo.EXPECT().GetLatestSummary(testUser.PreferredChannelID).Return(longSummary, nil)

	var sentParts []string
	for i, part := range expectedParts {
		withKeyboard := i == len(expectedParts)-1
		if withKeyboard {
			mockContext.EXPECT().Send(
				part,
				keyboard.GetStartKeyboard(),
				&tele.SendOptions{ParseMode: tele.ModeMarkdown},
			).DoAndReturn(func(what any, opts ...any) error {
				sentParts = append(sentParts, what.(string))
				return nil
			})
		} else {
			mockContext.EXPECT().Send(
				part,
				&tele.SendOptions{ParseMode: tele.ModeMarkdown},
			).DoAndReturn(func(what any, opts ...any) error {
				sentParts = append(sentParts, what.(string))
				return nil
			})
		}
	}

	handler := NewNewsHandler(mockSummaryRepo)
	if err := handler.Handle(mockContext); err != nil {
		t.Fatalf("Handle() error = %v", err)
	}

	if len(sentParts) != len(expectedParts) {
		t.Fatalf("expected %d sent parts, got %d", len(expectedParts), len(sentParts))
	}

	for i, part := range sentParts {
		if len([]rune(part)) > telegramutil.MaxMessageLength {
			t.Fatalf("part %d exceeds limit: %d runes", i, len([]rune(part)))
		}
	}

	if strings.Join(sentParts, "") != fullMessage {
		t.Fatal("sent parts do not reconstruct full summary message")
	}
}
