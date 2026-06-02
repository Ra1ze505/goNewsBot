package service

import (
	"context"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/Ra1ze505/goNewsBot/src/keyboard"
	mock_repository "github.com/Ra1ze505/goNewsBot/src/mocks/repository"
	mock_telebot "github.com/Ra1ze505/goNewsBot/src/mocks/telebot"
	"github.com/Ra1ze505/goNewsBot/src/repository"
	"github.com/Ra1ze505/goNewsBot/src/telegramutil"
	gomock "go.uber.org/mock/gomock"
	tele "gopkg.in/telebot.v4"
)

func TestMailingService_SendMailings(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockUserRepo := mock_repository.NewMockUserRepositoryInterface(ctrl)
	mockRateRepo := mock_repository.NewMockRateRepositoryInterface(ctrl)
	mockSummaryRepo := mock_repository.NewMockSummaryRepositoryInterface(ctrl)
	mockWeatherRepo := mock_repository.NewMockWeatherRepositoryInterface(ctrl)
	mockBot := mock_telebot.NewMockBot(ctrl)

	testUser := &repository.User{
		ID:                 &[]int{1}[0],
		Username:           &[]string{"test_user"}[0],
		ChatID:             123,
		City:               "Москва",
		Timezone:           "3",
		MailingTime:        time.Date(0, 0, 0, 10, 0, 0, 0, time.UTC),
		PreferredChannelID: 1429590454,
	}

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

	mockRateRepo.EXPECT().GetRates().Return(testRates, nil)
	mockWeatherRepo.EXPECT().GetWeatherByCity(testUser.City).Return(testWeather, nil)
	mockSummaryRepo.EXPECT().GetLatestSummary(testUser.PreferredChannelID).Return(testSummary, nil)
	mockBot.EXPECT().Send(gomock.Any(), gomock.Any(), gomock.Any()).Return(&tele.Message{}, nil)

	service := NewMailingService(
		mockUserRepo,
		mockRateRepo,
		mockSummaryRepo,
		mockWeatherRepo,
		mockBot,
	)

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	go service.StartMailingService(ctx)

	service.mailingChan <- testUser

	<-ctx.Done()
}

func TestMailingService_SendMailingsLongSummary(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockUserRepo := mock_repository.NewMockUserRepositoryInterface(ctrl)
	mockRateRepo := mock_repository.NewMockRateRepositoryInterface(ctrl)
	mockSummaryRepo := mock_repository.NewMockSummaryRepositoryInterface(ctrl)
	mockWeatherRepo := mock_repository.NewMockWeatherRepositoryInterface(ctrl)
	mockBot := mock_telebot.NewMockBot(ctrl)

	testUser := &repository.User{
		ID:                 &[]int{1}[0],
		Username:           &[]string{"test_user"}[0],
		ChatID:             123,
		City:               "Москва",
		Timezone:           "3",
		MailingTime:        time.Date(0, 0, 0, 10, 0, 0, 0, time.UTC),
		PreferredChannelID: 1429590454,
	}

	testRates := &repository.Rates{
		USD: repository.CurrencyRate{Value: 90.0, Previous: 89.0},
		EUR: repository.CurrencyRate{Value: 100.0, Previous: 99.0},
	}

	testWeather := &repository.WeatherResponse{
		Main: repository.MainResponse{Temp: 20.0},
		City: "Москва",
		Weather: []repository.WResponse{
			{Desc: "ясно"},
		},
	}

	testSummary := &repository.Summary{
		Summary:   strings.Repeat("a", 4097),
		CreatedAt: time.Date(2026, 6, 3, 10, 0, 0, 0, time.UTC),
	}

	weatherMsg := fmt.Sprintf("Погода в городе: %s\n%.1f градусов\n%s",
		testWeather.City, testWeather.Main.Temp, testWeather.Weather[0].Desc)
	ratesMsg := fmt.Sprintf("**Курс валют на сегодня**\n"+
		"Доллар: %.2f ₽ (изменение: %.2f%%)\n"+
		"Евро: %.2f ₽ (изменение: %.2f%%)",
		testRates.USD.Value,
		((testRates.USD.Value-testRates.USD.Previous)/testRates.USD.Previous)*100,
		testRates.EUR.Value,
		((testRates.EUR.Value-testRates.EUR.Previous)/testRates.EUR.Previous)*100)
	newsMsg := testSummary.GetFormattedSummary()
	fullMessage := fmt.Sprintf("Ежедневная рассылка:\n\n%s\n\n%s\n\n%s", weatherMsg, ratesMsg, newsMsg)
	expectedParts := telegramutil.SplitMessage(fullMessage)

	mockRateRepo.EXPECT().GetRates().Return(testRates, nil)
	mockWeatherRepo.EXPECT().GetWeatherByCity(testUser.City).Return(testWeather, nil)
	mockSummaryRepo.EXPECT().GetLatestSummary(testUser.PreferredChannelID).Return(testSummary, nil)

	var sentParts []string
	var lastOpts []interface{}
	for i, part := range expectedParts {
		withKeyboard := i == len(expectedParts)-1
		if withKeyboard {
			mockBot.EXPECT().Send(
				gomock.Any(),
				part,
				keyboard.GetStartKeyboard(),
				&tele.SendOptions{ParseMode: tele.ModeMarkdown},
			).DoAndReturn(func(to tele.Recipient, what interface{}, opts ...interface{}) (*tele.Message, error) {
				sentParts = append(sentParts, what.(string))
				lastOpts = opts
				return &tele.Message{}, nil
			})
		} else {
			mockBot.EXPECT().Send(
				gomock.Any(),
				part,
				&tele.SendOptions{ParseMode: tele.ModeMarkdown},
			).DoAndReturn(func(to tele.Recipient, what interface{}, opts ...interface{}) (*tele.Message, error) {
				sentParts = append(sentParts, what.(string))
				return &tele.Message{}, nil
			})
		}
	}

	service := NewMailingService(
		mockUserRepo,
		mockRateRepo,
		mockSummaryRepo,
		mockWeatherRepo,
		mockBot,
	)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go service.sendMailings(ctx)
	service.mailingChan <- testUser

	time.Sleep(100 * time.Millisecond)
	cancel()
	time.Sleep(50 * time.Millisecond)

	if len(sentParts) != len(expectedParts) {
		t.Fatalf("expected %d sent parts, got %d", len(expectedParts), len(sentParts))
	}

	if !strings.Contains(sentParts[0], "Ежедневная рассылка") {
		t.Fatalf("first part should contain mailing header, got %q", sentParts[0])
	}
	if !strings.Contains(sentParts[0], "Погода в городе: Москва") {
		t.Fatalf("first part should contain weather, got %q", sentParts[0])
	}
	if !strings.Contains(sentParts[0], "Курс валют на сегодня") {
		t.Fatalf("first part should contain rates, got %q", sentParts[0])
	}

	for i, part := range sentParts {
		if len([]rune(part)) > telegramutil.MaxMessageLength {
			t.Fatalf("part %d exceeds limit: %d runes", i, len([]rune(part)))
		}
	}

	if strings.Join(sentParts, "") != fullMessage {
		t.Fatal("sent parts do not reconstruct full mailing message")
	}

	if len(lastOpts) != 2 {
		t.Fatalf("expected keyboard on last part, got %d opts", len(lastOpts))
	}
	if _, ok := lastOpts[0].(*tele.ReplyMarkup); !ok {
		t.Fatalf("expected keyboard markup on last part, got %T", lastOpts[0])
	}
}

func TestMailingService_ErrorHandling(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockUserRepo := mock_repository.NewMockUserRepositoryInterface(ctrl)
	mockRateRepo := mock_repository.NewMockRateRepositoryInterface(ctrl)
	mockSummaryRepo := mock_repository.NewMockSummaryRepositoryInterface(ctrl)
	mockWeatherRepo := mock_repository.NewMockWeatherRepositoryInterface(ctrl)
	mockBot := mock_telebot.NewMockBot(ctrl)

	testUser := &repository.User{
		ID:                 &[]int{1}[0],
		Username:           &[]string{"test_user"}[0],
		ChatID:             123,
		City:               "Москва",
		Timezone:           "invalid",
		MailingTime:        time.Date(0, 0, 0, 10, 0, 0, 0, time.UTC),
		PreferredChannelID: 1429590454,
	}

	mockWeatherRepo.EXPECT().GetWeatherByCity(testUser.City).Return(nil, fmt.Errorf("test error"))

	service := NewMailingService(
		mockUserRepo,
		mockRateRepo,
		mockSummaryRepo,
		mockWeatherRepo,
		mockBot,
	)

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	go service.StartMailingService(ctx)

	service.mailingChan <- testUser

	<-ctx.Done()
}

func TestMailingService_EmptyUsers(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockUserRepo := mock_repository.NewMockUserRepositoryInterface(ctrl)
	mockRateRepo := mock_repository.NewMockRateRepositoryInterface(ctrl)
	mockSummaryRepo := mock_repository.NewMockSummaryRepositoryInterface(ctrl)
	mockWeatherRepo := mock_repository.NewMockWeatherRepositoryInterface(ctrl)
	mockBot := mock_telebot.NewMockBot(ctrl)

	service := NewMailingService(
		mockUserRepo,
		mockRateRepo,
		mockSummaryRepo,
		mockWeatherRepo,
		mockBot,
	)

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	go service.StartMailingService(ctx)

	<-ctx.Done()
}
