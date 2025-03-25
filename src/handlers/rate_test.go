package handlers

import (
	"testing"

	"github.com/Ra1ze505/goNewsBot/src/keyboard"
	mock_telebot "github.com/Ra1ze505/goNewsBot/src/mocks/telebot"
	"github.com/Ra1ze505/goNewsBot/src/repository"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"go.uber.org/mock/gomock"
)

type MockRateRepository struct {
	mock.Mock
}

func (m *MockRateRepository) SaveRate(rate *repository.Rate) error {
	args := m.Called(rate)
	return args.Error(0)
}

func (m *MockRateRepository) GetLatestRate() (*repository.Rate, error) {
	args := m.Called()
	return args.Get(0).(*repository.Rate), args.Error(1)
}

func (m *MockRateRepository) GetRates() (*repository.Rates, error) {
	args := m.Called()
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*repository.Rates), args.Error(1)
}

func TestRateHandler_Handle(t *testing.T) {
	mockRepo := new(MockRateRepository)

	// Test case: successful rate retrieval
	t.Run("successful rate retrieval", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		mockRepo.ExpectedCalls = nil
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

		mockRepo.On("GetRates").Return(rates, nil)
		mockContext.EXPECT().Send(expectedMessage, keyboard.GetStartKeyboard()).Return(nil)

		err := handler.Handle(mockContext)
		assert.NoError(t, err)
		mockRepo.AssertExpectations(t)
	})

	// Test case: error retrieving rates
	t.Run("error retrieving rates", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		mockRepo.ExpectedCalls = nil
		mockContext := mock_telebot.NewMockContext(ctrl)
		handler := NewRateHandler(mockRepo)

		errorMessage := "Извините, не удалось получить текущий курс валют. Попробуйте позже."
		mockRepo.On("GetRates").Return(nil, assert.AnError)
		mockContext.EXPECT().Send(errorMessage, keyboard.GetStartKeyboard()).Return(nil)

		err := handler.Handle(mockContext)
		assert.NoError(t, err)
		mockRepo.AssertExpectations(t)
	})
}
