package service

import (
	"context"
	"fmt"
	"testing"
	"time"

	mock_repository "github.com/Ra1ze505/goNewsBot/src/mocks/repository"
	mock_telebot "github.com/Ra1ze505/goNewsBot/src/mocks/telebot"
	"github.com/Ra1ze505/goNewsBot/src/repository"
	gomock "go.uber.org/mock/gomock"
	tele "gopkg.in/telebot.v4"
)

func TestMailingService_SendMailings(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	// Create mocks
	mockUserRepo := mock_repository.NewMockUserRepositoryInterface(ctrl)
	mockRateRepo := mock_repository.NewMockRateRepositoryInterface(ctrl)
	mockSummaryRepo := mock_repository.NewMockSummaryRepositoryInterface(ctrl)
	mockWeatherRepo := mock_repository.NewMockWeatherRepositoryInterface(ctrl)
	mockBot := mock_telebot.NewMockBot(ctrl)

	// Create test user
	testUser := &repository.User{
		ID:                 &[]int{1}[0],
		Username:           &[]string{"test_user"}[0],
		ChatID:             123,
		City:               "Москва",
		Timezone:           "3",
		MailingTime:        time.Date(0, 0, 0, 10, 0, 0, 0, time.UTC),
		PreferredChannelID: 1429590454,
	}

	// Create test data
	testRates := &repository.Rates{
		USD: repository.CurrencyRate{
			Value:    90.0,
			Previous: 89.0,
		},
		EUR: repository.CurrencyRate{
			Value:    100.0,
			Previous: 99.0,
		},
	}

	testWeather := &repository.WeatherResponse{
		Main: repository.MainResponse{
			Temp: 20.0,
		},
		City: "Москва",
		Weather: []repository.WResponse{
			{Desc: "ясно"},
		},
	}

	testSummary := &repository.Summary{
		Summary: "Тестовая новость",
	}

	// Set up expectations
	mockRateRepo.EXPECT().GetRates().Return(testRates, nil)
	mockWeatherRepo.EXPECT().GetWeatherByCity(testUser.City).Return(testWeather, nil)
	mockSummaryRepo.EXPECT().GetLatestSummary(testUser.PreferredChannelID).Return(testSummary, nil)
	mockBot.EXPECT().Send(gomock.Any(), gomock.Any(), gomock.Any()).Return(&tele.Message{}, nil)

	// Create service
	service := NewMailingService(
		mockUserRepo,
		mockRateRepo,
		mockSummaryRepo,
		mockWeatherRepo,
		mockBot,
	)

	// Create context with timeout
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	// Start service
	go service.StartMailingService(ctx)

	// Send a test user directly to the mailing channel
	service.mailingChan <- testUser

	// Wait for context to be done
	<-ctx.Done()
}

func TestMailingService_ErrorHandling(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	// Create mocks
	mockUserRepo := mock_repository.NewMockUserRepositoryInterface(ctrl)
	mockRateRepo := mock_repository.NewMockRateRepositoryInterface(ctrl)
	mockSummaryRepo := mock_repository.NewMockSummaryRepositoryInterface(ctrl)
	mockWeatherRepo := mock_repository.NewMockWeatherRepositoryInterface(ctrl)
	mockBot := mock_telebot.NewMockBot(ctrl)

	// Create test user with invalid timezone
	testUser := &repository.User{
		ID:                 &[]int{1}[0],
		Username:           &[]string{"test_user"}[0],
		ChatID:             123,
		City:               "Москва",
		Timezone:           "invalid",
		MailingTime:        time.Date(0, 0, 0, 10, 0, 0, 0, time.UTC),
		PreferredChannelID: 1429590454,
	}

	// Set up expectations
	mockWeatherRepo.EXPECT().GetWeatherByCity(testUser.City).Return(nil, fmt.Errorf("test error"))

	// Create service
	service := NewMailingService(
		mockUserRepo,
		mockRateRepo,
		mockSummaryRepo,
		mockWeatherRepo,
		mockBot,
	)

	// Create context with timeout
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	// Start service
	go service.StartMailingService(ctx)

	// Send a test user directly to the mailing channel
	service.mailingChan <- testUser

	// Wait for context to be done
	<-ctx.Done()
}

func TestMailingService_EmptyUsers(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	// Create mocks
	mockUserRepo := mock_repository.NewMockUserRepositoryInterface(ctrl)
	mockRateRepo := mock_repository.NewMockRateRepositoryInterface(ctrl)
	mockSummaryRepo := mock_repository.NewMockSummaryRepositoryInterface(ctrl)
	mockWeatherRepo := mock_repository.NewMockWeatherRepositoryInterface(ctrl)
	mockBot := mock_telebot.NewMockBot(ctrl)

	// Create service
	service := NewMailingService(
		mockUserRepo,
		mockRateRepo,
		mockSummaryRepo,
		mockWeatherRepo,
		mockBot,
	)

	// Create context with timeout
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	// Start service
	go service.StartMailingService(ctx)

	// Wait for context to be done
	<-ctx.Done()
}
