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
	"go.uber.org/mock/gomock"
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
		expectedMsg string
		expectedErr error
		setupMocks  func()
	}{
		{
			name:        "Success case",
			summary:     testSummary,
			summaryErr:  nil,
			expectedMsg: "Последние новости:\n\nTest summary content",
			expectedErr: nil,
			setupMocks: func() {
				mockContext.EXPECT().Get("user").Return(testUser)
				mockSummaryRepo.EXPECT().GetLatestSummary(testUser.PreferredChannelID).Return(testSummary, nil)
				mockContext.EXPECT().Send("Последние новости:\n\nTest summary content", keyboard.GetStartKeyboard()).Return(nil)
			},
		},
		{
			name:        "No news available",
			summary:     nil,
			summaryErr:  nil,
			expectedMsg: "Новостей пока нет. Проверьте позже.",
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
			expectedMsg: "Произошла ошибка при получении новостей. Попробуйте позже.",
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
			expectedMsg: "",
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
				CreatedAt: time.Now(),
			},
			summaryErr:  nil,
			expectedMsg: "Суммарная длина новостей превышает 4096 символов. Воспользуйтесь кнопкой 'Написать нам' и сообщите о проблеме.",
			expectedErr: nil,
			setupMocks: func() {
				mockContext.EXPECT().Get("user").Return(testUser)
				mockSummaryRepo.EXPECT().GetLatestSummary(testUser.PreferredChannelID).Return(&repository.Summary{
					ID:        1,
					ChannelID: 123,
					Summary:   strings.Repeat("a", 4097),
					CreatedAt: time.Now(),
				}, nil)
				mockContext.EXPECT().Send("Суммарная длина новостей превышает 4096 символов. Воспользуйтесь кнопкой 'Написать нам' и сообщите о проблеме.", keyboard.GetStartKeyboard()).Return(nil)
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
