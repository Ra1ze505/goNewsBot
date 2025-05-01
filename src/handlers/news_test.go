package handlers

import (
	"errors"
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

	// Создаем моки
	mockSummaryRepo := mock_repository.NewMockSummaryRepositoryInterface(ctrl)
	mockContext := mock_telebot.NewMockContext(ctrl)

	// Создаем тестовые данные
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
				mockSummaryRepo.EXPECT().GetLatestSummary().Return(testSummary, nil)
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
				mockSummaryRepo.EXPECT().GetLatestSummary().Return(nil, nil)
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
				mockSummaryRepo.EXPECT().GetLatestSummary().Return(nil, errors.New("database error"))
				mockContext.EXPECT().Send("Произошла ошибка при получении новостей. Попробуйте позже.", keyboard.GetStartKeyboard()).Return(nil)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Настраиваем моки
			tt.setupMocks()

			// Создаем хендлер
			handler := NewNewsHandler(mockSummaryRepo)

			// Вызываем тестируемый метод
			err := handler.Handle(mockContext)

			// Проверяем результаты
			if err != tt.expectedErr {
				t.Errorf("Handle() error = %v, expectedErr %v", err, tt.expectedErr)
			}
		})
	}
}
