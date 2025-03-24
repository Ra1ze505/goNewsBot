package middleware_test

import (
	"errors"
	"testing"

	"github.com/Ra1ze505/goNewsBot/src/middleware"
	"github.com/Ra1ze505/goNewsBot/src/mocks/repository"
	"github.com/Ra1ze505/goNewsBot/src/mocks/telebot"
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
	mockRepo.EXPECT().CreateOrUpdateUser(gomock.Any()).Return(nil)

	mockUser := &tele.User{
		ID:       123,
		Username: "test_user",
	}
	mockContext := mock_telebot.NewMockContext(ctrl)
	mockContext.EXPECT().Sender().AnyTimes().Return(mockUser)

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
	mockRepo.EXPECT().CreateOrUpdateUser(gomock.Any()).Return(expectedError)

	mockUser := &tele.User{
		ID:       456,
		Username: "error_user",
	}
	mockContext := mock_telebot.NewMockContext(ctrl)
	mockContext.EXPECT().Sender().AnyTimes().Return(mockUser)
	mockContext.EXPECT().Send(gomock.Any())

	m := middleware.CreateOrUpdateUser(mockRepo)
	f := m(nextMock)
	err := f(mockContext)
	if err == nil {
		t.Error("Expected error but got nil")
	}
}

