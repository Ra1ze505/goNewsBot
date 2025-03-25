package middleware_test

import (
	"errors"
	"testing"
	"time"

	"github.com/Ra1ze505/goNewsBot/src/middleware"
	mock_repository "github.com/Ra1ze505/goNewsBot/src/mocks/repository"
	mock_telebot "github.com/Ra1ze505/goNewsBot/src/mocks/telebot"
	"github.com/Ra1ze505/goNewsBot/src/repository"
	gomock "go.uber.org/mock/gomock"
	tele "gopkg.in/telebot.v4"
)

func nextMock(c tele.Context) error {
	return nil
}

func TestHelloHandle_CreateOrUpdateUser(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockRepo := mock_repository.NewMockUserRepositoryInterface(ctrl)
	mockUser := &repository.User{
		ID:          &[]int{123}[0],
		Username:    &[]string{"test_user"}[0],
		ChatID:      123,
		City:        "Москва",
		Timezone:    "3",
		MailingTime: time.Date(0, 0, 0, 10, 0, 0, 0, time.Local),
	}
	mockRepo.EXPECT().CreateOrUpdateUser(gomock.Any()).Return(mockUser, nil)

	mockTeleUser := &tele.User{
		ID:       123,
		Username: "test_user",
	}
	mockContext := mock_telebot.NewMockContext(ctrl)
	mockContext.EXPECT().Sender().AnyTimes().Return(mockTeleUser)
	mockContext.EXPECT().Set("user", mockUser)
	mockContext.EXPECT().Callback().AnyTimes().Return(nil)

	m := middleware.CreateOrUpdateUser(mockRepo)
	f := m(nextMock)
	err := f(mockContext)
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
}

func TestHelloHandle_ErrorFromRepository(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockRepo := mock_repository.NewMockUserRepositoryInterface(ctrl)
	expectedError := errors.New("failed to create or update user")
	mockRepo.EXPECT().CreateOrUpdateUser(gomock.Any()).Return(nil, expectedError)

	mockTeleUser := &tele.User{
		ID:       456,
		Username: "error_user",
	}
	mockContext := mock_telebot.NewMockContext(ctrl)
	mockContext.EXPECT().Sender().AnyTimes().Return(mockTeleUser)
	mockContext.EXPECT().Send("Что-то пошло не так :(\nПопробуй позже")
	mockContext.EXPECT().Callback().AnyTimes().Return(nil)

	m := middleware.CreateOrUpdateUser(mockRepo)
	f := m(nextMock)
	err := f(mockContext)
	if err != expectedError {
		t.Errorf("Expected error %v but got %v", expectedError, err)
	}
}
