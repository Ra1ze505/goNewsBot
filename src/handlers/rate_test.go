package handlers

import (
	"testing"

	"github.com/Ra1ze505/goNewsBot/src/keyboard"
	mock_repository "github.com/Ra1ze505/goNewsBot/src/mocks/repository"
	mock_telebot "github.com/Ra1ze505/goNewsBot/src/mocks/telebot"
	"github.com/Ra1ze505/goNewsBot/src/repository"
	"github.com/stretchr/testify/assert"
	"go.uber.org/mock/gomock"
)

func TestRateHandler_Handle(t *testing.T) {
	// Test case: successful rate retrieval
	t.Run("successful rate retrieval", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		mockRepo := mock_repository.NewMockRateRepositoryInterface(ctrl)
		mockContext := mock_telebot.NewMockContext(ctrl)
		handler := NewRateHandler(mockRepo)

		rates := &repository.Rates{
			USD: repository.CurrencyRate{
				Value:    90.0,
				Previous: 89.0,
			},
			EUR: repository.CurrencyRate{
				Value:    100.0,
				Previous: 99.0,
			},
		}

		expectedMessage := "**Курс валют на сегодня**\n" +
			"Доллар: 90.00 ₽ (изменение: 1.12%)\n" +
			"Евро: 100.00 ₽ (изменение: 1.01%)"

		mockRepo.EXPECT().GetRates().Return(rates, nil)
		mockContext.EXPECT().Send(expectedMessage, keyboard.GetStartKeyboard()).Return(nil)

		err := handler.Handle(mockContext)
		assert.NoError(t, err)
	})

	// Test case: error retrieving rates
	t.Run("error retrieving rates", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		mockRepo := mock_repository.NewMockRateRepositoryInterface(ctrl)
		mockContext := mock_telebot.NewMockContext(ctrl)
		handler := NewRateHandler(mockRepo)

		errorMessage := "Извините, не удалось получить текущий курс валют. Попробуйте позже."
		mockRepo.EXPECT().GetRates().Return(nil, assert.AnError)
		mockContext.EXPECT().Send(errorMessage, keyboard.GetStartKeyboard()).Return(nil)

		err := handler.Handle(mockContext)
		assert.NoError(t, err)
	})
}
