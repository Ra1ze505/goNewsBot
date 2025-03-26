package handlers_test

import (
	"errors"
	"testing"

	"github.com/Ra1ze505/goNewsBot/src/handlers"
	"github.com/Ra1ze505/goNewsBot/src/keyboard"
	mock_repository "github.com/Ra1ze505/goNewsBot/src/mocks/repository"
	mock_telebot "github.com/Ra1ze505/goNewsBot/src/mocks/telebot"
	"github.com/Ra1ze505/goNewsBot/src/repository"
	gomock "go.uber.org/mock/gomock"
)

func TestChangeCityHandler_Handle(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockUserRepo := mock_repository.NewMockUserRepositoryInterface(ctrl)
	mockWeatherRepo := mock_repository.NewMockWeatherRepositoryInterface(ctrl)
	stateStorage := handlers.NewStateStorage()

	handler := handlers.NewChangeCityHandler(mockUserRepo, mockWeatherRepo, stateStorage)

	mockUser := &repository.User{
		ID:       &[]int{123}[0],
		Username: &[]string{"test_user"}[0],
		ChatID:   123,
		City:     "Москва",
		Timezone: "3",
	}

	mockContext := mock_telebot.NewMockContext(ctrl)
	mockContext.EXPECT().Get("user").Return(mockUser)
	mockContext.EXPECT().Send("Ваш город сейчас: Москва\nВыберите город из списка или напишите свой", keyboard.GetCitySelectionKeyboard())

	err := handler.Handle(mockContext)
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}

	// Verify state was set
	state := stateStorage.GetState(mockUser.ChatID)
	if state == nil || !state.ChangingCity {
		t.Error("Expected state to be set with ChangingCity=true")
	}
}

func TestChangeCityHandler_HandleCityInput_Cancel(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockUserRepo := mock_repository.NewMockUserRepositoryInterface(ctrl)
	mockWeatherRepo := mock_repository.NewMockWeatherRepositoryInterface(ctrl)
	stateStorage := handlers.NewStateStorage()

	handler := handlers.NewChangeCityHandler(mockUserRepo, mockWeatherRepo, stateStorage)

	mockUser := &repository.User{
		ID:       &[]int{123}[0],
		Username: &[]string{"test_user"}[0],
		ChatID:   123,
		City:     "Москва",
		Timezone: "3",
	}

	// Set initial state
	stateStorage.SetState(mockUser.ChatID, &handlers.UserState{ChangingCity: true})

	mockContext := mock_telebot.NewMockContext(ctrl)
	mockContext.EXPECT().Get("user").Return(mockUser)
	mockContext.EXPECT().Text().Return("Отмена")
	mockContext.EXPECT().Send("Город не изменен", keyboard.GetStartKeyboard())

	err := handler.HandleCityInput(mockContext)
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}

	// Verify state was cleared
	state := stateStorage.GetState(mockUser.ChatID)
	if state != nil {
		t.Error("Expected state to be cleared")
	}
}

func TestChangeCityHandler_HandleCityInput_InvalidCity(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockUserRepo := mock_repository.NewMockUserRepositoryInterface(ctrl)
	mockWeatherRepo := mock_repository.NewMockWeatherRepositoryInterface(ctrl)
	stateStorage := handlers.NewStateStorage()

	handler := handlers.NewChangeCityHandler(mockUserRepo, mockWeatherRepo, stateStorage)

	mockUser := &repository.User{
		ID:       &[]int{123}[0],
		Username: &[]string{"test_user"}[0],
		ChatID:   123,
		City:     "Москва",
		Timezone: "3",
	}

	// Set initial state
	stateStorage.SetState(mockUser.ChatID, &handlers.UserState{ChangingCity: true})

	mockContext := mock_telebot.NewMockContext(ctrl)
	mockContext.EXPECT().Get("user").AnyTimes().Return(mockUser)
	mockContext.EXPECT().Text().AnyTimes().Return("Несуществующий город")
	mockWeatherRepo.EXPECT().GetWeatherByCity("Несуществующий город").AnyTimes().Return(nil, errors.New("city not found"))
	mockContext.EXPECT().Send("Некорректный город\nПопробуйте еще раз", keyboard.GetStartKeyboard()).AnyTimes()

	err := handler.HandleCityInput(mockContext)
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}

	// Verify state was not cleared
	state := stateStorage.GetState(mockUser.ChatID)
	if state == nil || !state.ChangingCity {
		t.Error("Expected state to remain unchanged")
	}
}

func TestChangeCityHandler_HandleCityInput_Success(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockUserRepo := mock_repository.NewMockUserRepositoryInterface(ctrl)
	mockWeatherRepo := mock_repository.NewMockWeatherRepositoryInterface(ctrl)
	stateStorage := handlers.NewStateStorage()

	handler := handlers.NewChangeCityHandler(mockUserRepo, mockWeatherRepo, stateStorage)

	mockUser := &repository.User{
		ID:       &[]int{123}[0],
		Username: &[]string{"test_user"}[0],
		ChatID:   123,
		City:     "Москва",
		Timezone: "3",
	}

	// Set initial state
	stateStorage.SetState(mockUser.ChatID, &handlers.UserState{ChangingCity: true})

	mockContext := mock_telebot.NewMockContext(ctrl)
	mockContext.EXPECT().Get("user").AnyTimes().Return(mockUser)
	mockContext.EXPECT().Text().AnyTimes().Return("Санкт-Петербург")

	weather := &repository.WeatherResponse{
		City:     "Санкт-Петербург",
		Timezone: 10800, // UTC+3
	}
	mockWeatherRepo.EXPECT().GetWeatherByCity("Санкт-Петербург").AnyTimes().Return(weather, nil)
	mockUserRepo.EXPECT().UpdateUserCityAndTimezone(mockUser.ID, "Санкт-Петербург", "3").AnyTimes().Return(nil)
	mockContext.EXPECT().Send("Город изменен на Санкт-Петербург", keyboard.GetStartKeyboard()).AnyTimes()

	err := handler.HandleCityInput(mockContext)
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}

	// Verify state was cleared
	state := stateStorage.GetState(mockUser.ChatID)
	if state != nil {
		t.Error("Expected state to be cleared")
	}
}
