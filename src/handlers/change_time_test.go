package handlers_test

import (
	"errors"
	"testing"
	"time"

	"github.com/Ra1ze505/goNewsBot/src/handlers"
	"github.com/Ra1ze505/goNewsBot/src/keyboard"
	mock_repository "github.com/Ra1ze505/goNewsBot/src/mocks/repository"
	mock_telebot "github.com/Ra1ze505/goNewsBot/src/mocks/telebot"
	"github.com/Ra1ze505/goNewsBot/src/repository"
	gomock "go.uber.org/mock/gomock"
)

func TestChangeTimeHandler_Handle(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockUserRepo := mock_repository.NewMockUserRepositoryInterface(ctrl)
	stateStorage := handlers.NewStateStorage()

	handler := handlers.NewChangeTimeHandler(mockUserRepo, stateStorage)

	mockUser := &repository.User{
		ID:          &[]int{123}[0],
		Username:    &[]string{"test_user"}[0],
		ChatID:      123,
		MailingTime: time.Date(0, 0, 0, 10, 0, 0, 0, time.Local),
	}

	mockContext := mock_telebot.NewMockContext(ctrl)
	mockContext.EXPECT().Get("user").Return(mockUser)
	mockContext.EXPECT().Send("Ваше текущее время рассылки: 10:00\n\nВыберите время из предложенных или введите свое в формате ЧЧ:ММ (например, 09:00):", keyboard.GetTimeSelectionKeyboard())

	err := handler.Handle(mockContext)
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}

	// Verify state was set
	state := stateStorage.GetState(mockUser.ChatID)
	if state == nil || !state.ChangingTime {
		t.Error("Expected state to be set with ChangingTime=true")
	}
}

func TestChangeTimeHandler_HandleTimeInput_Cancel(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockUserRepo := mock_repository.NewMockUserRepositoryInterface(ctrl)
	stateStorage := handlers.NewStateStorage()

	handler := handlers.NewChangeTimeHandler(mockUserRepo, stateStorage)

	mockUser := &repository.User{
		ID:          &[]int{123}[0],
		Username:    &[]string{"test_user"}[0],
		ChatID:      123,
		MailingTime: time.Date(0, 0, 0, 10, 0, 0, 0, time.Local),
	}

	// Set initial state
	stateStorage.SetState(mockUser.ChatID, &handlers.UserState{ChangingTime: true})

	mockContext := mock_telebot.NewMockContext(ctrl)
	mockContext.EXPECT().Get("user").Return(mockUser)
	mockContext.EXPECT().Text().Return(keyboard.CancelBtn.Text)
	mockContext.EXPECT().Send("Время не изменено", keyboard.GetStartKeyboard())

	err := handler.HandleTimeInput(mockContext)
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}

	// Verify state was cleared
	state := stateStorage.GetState(mockUser.ChatID)
	if state != nil {
		t.Error("Expected state to be cleared")
	}
}

func TestChangeTimeHandler_HandleTimeInput_InvalidFormat(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockUserRepo := mock_repository.NewMockUserRepositoryInterface(ctrl)
	stateStorage := handlers.NewStateStorage()

	handler := handlers.NewChangeTimeHandler(mockUserRepo, stateStorage)

	mockUser := &repository.User{
		ID:          &[]int{123}[0],
		Username:    &[]string{"test_user"}[0],
		ChatID:      123,
		MailingTime: time.Date(0, 0, 0, 10, 0, 0, 0, time.Local),
	}

	// Set initial state
	stateStorage.SetState(mockUser.ChatID, &handlers.UserState{ChangingTime: true})

	mockContext := mock_telebot.NewMockContext(ctrl)
	mockContext.EXPECT().Get("user").AnyTimes().Return(mockUser)
	mockContext.EXPECT().Text().AnyTimes().Return("invalid_time")
	mockContext.EXPECT().Send("Некорректный формат времени. Используйте формат ЧЧ:ММ (например, 09:00)", keyboard.GetStartKeyboard()).AnyTimes()

	err := handler.HandleTimeInput(mockContext)
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}

	// Verify state was not cleared
	state := stateStorage.GetState(mockUser.ChatID)
	if state == nil || !state.ChangingTime {
		t.Error("Expected state to remain unchanged")
	}
}

func TestChangeTimeHandler_HandleTimeInput_InvalidHour(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockUserRepo := mock_repository.NewMockUserRepositoryInterface(ctrl)
	stateStorage := handlers.NewStateStorage()

	handler := handlers.NewChangeTimeHandler(mockUserRepo, stateStorage)

	mockUser := &repository.User{
		ID:          &[]int{123}[0],
		Username:    &[]string{"test_user"}[0],
		ChatID:      123,
		MailingTime: time.Date(0, 0, 0, 10, 0, 0, 0, time.Local),
	}

	// Set initial state
	stateStorage.SetState(mockUser.ChatID, &handlers.UserState{ChangingTime: true})

	mockContext := mock_telebot.NewMockContext(ctrl)
	mockContext.EXPECT().Get("user").AnyTimes().Return(mockUser)
	mockContext.EXPECT().Text().AnyTimes().Return("25:00")
	mockContext.EXPECT().Send("Некорректный час. Используйте значения от 00 до 23", keyboard.GetStartKeyboard()).AnyTimes()

	err := handler.HandleTimeInput(mockContext)
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}

	// Verify state was not cleared
	state := stateStorage.GetState(mockUser.ChatID)
	if state == nil || !state.ChangingTime {
		t.Error("Expected state to remain unchanged")
	}
}

func TestChangeTimeHandler_HandleTimeInput_InvalidMinute(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockUserRepo := mock_repository.NewMockUserRepositoryInterface(ctrl)
	stateStorage := handlers.NewStateStorage()

	handler := handlers.NewChangeTimeHandler(mockUserRepo, stateStorage)

	mockUser := &repository.User{
		ID:          &[]int{123}[0],
		Username:    &[]string{"test_user"}[0],
		ChatID:      123,
		MailingTime: time.Date(0, 0, 0, 10, 0, 0, 0, time.Local),
	}

	// Set initial state
	stateStorage.SetState(mockUser.ChatID, &handlers.UserState{ChangingTime: true})

	mockContext := mock_telebot.NewMockContext(ctrl)
	mockContext.EXPECT().Get("user").AnyTimes().Return(mockUser)
	mockContext.EXPECT().Text().AnyTimes().Return("10:60")
	mockContext.EXPECT().Send("Некорректные минуты. Используйте значения от 00 до 59", keyboard.GetStartKeyboard()).AnyTimes()

	err := handler.HandleTimeInput(mockContext)
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}

	// Verify state was not cleared
	state := stateStorage.GetState(mockUser.ChatID)
	if state == nil || !state.ChangingTime {
		t.Error("Expected state to remain unchanged")
	}
}

func TestChangeTimeHandler_HandleTimeInput_Success(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockUserRepo := mock_repository.NewMockUserRepositoryInterface(ctrl)
	stateStorage := handlers.NewStateStorage()

	handler := handlers.NewChangeTimeHandler(mockUserRepo, stateStorage)

	mockUser := &repository.User{
		ID:          &[]int{123}[0],
		Username:    &[]string{"test_user"}[0],
		ChatID:      123,
		MailingTime: time.Date(0, 0, 0, 10, 0, 0, 0, time.Local),
	}

	// Set initial state
	stateStorage.SetState(mockUser.ChatID, &handlers.UserState{ChangingTime: true})

	mockContext := mock_telebot.NewMockContext(ctrl)
	mockContext.EXPECT().Get("user").AnyTimes().Return(mockUser)
	mockContext.EXPECT().Text().AnyTimes().Return("09:00")

	expectedTime := time.Date(0, 0, 0, 9, 0, 0, 0, time.Local)
	mockUserRepo.EXPECT().UpdateUserMailingTime(mockUser.ID, expectedTime).AnyTimes().Return(nil)
	mockContext.EXPECT().Send("Время рассылки изменено на 09:00", keyboard.GetStartKeyboard()).AnyTimes()

	err := handler.HandleTimeInput(mockContext)
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}

	// Verify state was cleared
	state := stateStorage.GetState(mockUser.ChatID)
	if state != nil {
		t.Error("Expected state to be cleared")
	}
}

func TestChangeTimeHandler_HandleTimeInput_RepositoryError(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockUserRepo := mock_repository.NewMockUserRepositoryInterface(ctrl)
	stateStorage := handlers.NewStateStorage()

	handler := handlers.NewChangeTimeHandler(mockUserRepo, stateStorage)

	mockUser := &repository.User{
		ID:          &[]int{123}[0],
		Username:    &[]string{"test_user"}[0],
		ChatID:      123,
		MailingTime: time.Date(0, 0, 0, 10, 0, 0, 0, time.Local),
	}

	// Set initial state
	stateStorage.SetState(mockUser.ChatID, &handlers.UserState{ChangingTime: true})

	mockContext := mock_telebot.NewMockContext(ctrl)
	mockContext.EXPECT().Get("user").AnyTimes().Return(mockUser)
	mockContext.EXPECT().Text().AnyTimes().Return("09:00")

	expectedTime := time.Date(0, 0, 0, 9, 0, 0, 0, time.Local)
	expectedError := errors.New("database error")
	mockUserRepo.EXPECT().UpdateUserMailingTime(mockUser.ID, expectedTime).AnyTimes().Return(expectedError)

	err := handler.HandleTimeInput(mockContext)
	if err == nil {
		t.Error("Expected error from repository")
	}
	if err.Error() != "failed to update user mailing time: database error" {
		t.Errorf("Unexpected error message: %v", err)
	}

	// Verify state was not cleared
	state := stateStorage.GetState(mockUser.ChatID)
	if state == nil || !state.ChangingTime {
		t.Error("Expected state to remain unchanged")
	}
}
