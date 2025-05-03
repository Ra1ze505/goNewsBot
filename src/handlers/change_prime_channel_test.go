package handlers_test

import (
	"fmt"
	"testing"

	"github.com/Ra1ze505/goNewsBot/src/config"
	"github.com/Ra1ze505/goNewsBot/src/handlers"
	"github.com/Ra1ze505/goNewsBot/src/keyboard"
	mock_repository "github.com/Ra1ze505/goNewsBot/src/mocks/repository"
	mock_telebot "github.com/Ra1ze505/goNewsBot/src/mocks/telebot"
	"github.com/Ra1ze505/goNewsBot/src/repository"
	gomock "go.uber.org/mock/gomock"
	tele "gopkg.in/telebot.v4"
)

func TestChangePrimeChannelHandler_Handle(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockUserRepo := mock_repository.NewMockUserRepositoryInterface(ctrl)
	handler := handlers.NewChangePrimeChannelHandler(mockUserRepo)

	mockUser := &repository.User{
		ID:                 &[]int{123}[0],
		PreferredChannelID: 1,
	}

	mockContext := mock_telebot.NewMockContext(ctrl)

	config.Channels = map[int64]string{
		1: "Test Channel 1",
		2: "Test Channel 2",
	}

	gomock.InOrder(
		mockContext.EXPECT().Get("user").Return(mockUser),
		mockContext.EXPECT().Send(
			fmt.Sprintf("Ваш текущий новостной канал: %s\n\nВыберите новый канал:", config.Channels[mockUser.PreferredChannelID]),
			gomock.Any(),
		).Return(nil),
	)

	err := handler.Handle(mockContext)
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
}

func TestChangePrimeChannelHandler_HandleChannelSelection_Success(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockUserRepo := mock_repository.NewMockUserRepositoryInterface(ctrl)
	handler := handlers.NewChangePrimeChannelHandler(mockUserRepo)

	mockUser := &repository.User{
		ID:                 &[]int{123}[0],
		PreferredChannelID: 1,
	}

	mockContext := mock_telebot.NewMockContext(ctrl)

	config.Channels = map[int64]string{
		1: "Test Channel 1",
		2: "Test Channel 2",
	}

	mockContext.EXPECT().Get("user").Return(mockUser)
	mockContext.EXPECT().Callback().Return(&tele.Callback{Data: "channel_2"}).AnyTimes()
	mockUserRepo.EXPECT().UpdatePreferredChannel(mockUser.ID, int64(2)).Return(nil)
	mockContext.EXPECT().Send(
		fmt.Sprintf("Новостной канал изменен на: %s", config.Channels[2]),
		gomock.Any(),
	).Return(nil)

	err := handler.HandleChannelSelection(mockContext)
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
}

func TestChangePrimeChannelHandler_HandleChannelSelection_Cancel(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockUserRepo := mock_repository.NewMockUserRepositoryInterface(ctrl)
	handler := handlers.NewChangePrimeChannelHandler(mockUserRepo)

	mockUser := &repository.User{
		ID:                 &[]int{123}[0],
		PreferredChannelID: 1,
	}

	mockContext := mock_telebot.NewMockContext(ctrl)

	mockContext.EXPECT().Get("user").Return(mockUser)
	mockContext.EXPECT().Callback().Return(&tele.Callback{Data: keyboard.CancelBtn.Data}).AnyTimes()
	mockContext.EXPECT().Send("Выбор канала отменен", gomock.Any()).Return(nil)

	err := handler.HandleChannelSelection(mockContext)
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
}

func TestChangePrimeChannelHandler_HandleChannelSelection_InvalidFormat(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockUserRepo := mock_repository.NewMockUserRepositoryInterface(ctrl)
	handler := handlers.NewChangePrimeChannelHandler(mockUserRepo)

	mockUser := &repository.User{
		ID:                 &[]int{123}[0],
		PreferredChannelID: 1,
	}

	mockContext := mock_telebot.NewMockContext(ctrl)

	mockContext.EXPECT().Get("user").Return(mockUser)
	mockContext.EXPECT().Callback().Return(&tele.Callback{Data: "invalid_format"}).AnyTimes()

	err := handler.HandleChannelSelection(mockContext)
	if err == nil {
		t.Error("Expected error for invalid format")
	}
}

func TestChangePrimeChannelHandler_HandleChannelSelection_UserNotFound(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockUserRepo := mock_repository.NewMockUserRepositoryInterface(ctrl)
	handler := handlers.NewChangePrimeChannelHandler(mockUserRepo)

	mockContext := mock_telebot.NewMockContext(ctrl)
	mockContext.EXPECT().Get("user").Return(nil)

	err := handler.HandleChannelSelection(mockContext)
	if err == nil {
		t.Error("Expected error for user not found")
	}
}
