package service

import (
	"context"
	"fmt"
	"strconv"
	"time"

	"github.com/Ra1ze505/goNewsBot/src/keyboard"
	"github.com/Ra1ze505/goNewsBot/src/repository"
	log "github.com/sirupsen/logrus"
	tele "gopkg.in/telebot.v4"
)

// BotSender is a minimal interface that defines only the methods we need from tele.Bot
type BotSender interface {
	Send(to tele.Recipient, what interface{}, opts ...interface{}) (*tele.Message, error)
}

type MailingService struct {
	userRepo    repository.UserRepositoryInterface
	rateRepo    repository.RateRepositoryInterface
	summaryRepo repository.SummaryRepositoryInterface
	weatherRepo repository.WeatherRepositoryInterface
	bot         BotSender
	mailingChan chan *repository.User
}

func NewMailingService(
	userRepo repository.UserRepositoryInterface,
	rateRepo repository.RateRepositoryInterface,
	summaryRepo repository.SummaryRepositoryInterface,
	weatherRepo repository.WeatherRepositoryInterface,
	bot BotSender,
) *MailingService {
	return &MailingService{
		userRepo:    userRepo,
		rateRepo:    rateRepo,
		summaryRepo: summaryRepo,
		weatherRepo: weatherRepo,
		bot:         bot,
		mailingChan: make(chan *repository.User, 100),
	}
}

func (s *MailingService) StartMailingService(ctx context.Context) {
	go s.scheduleMailings(ctx)

	go s.sendMailings(ctx)
}

func (s *MailingService) scheduleMailings(ctx context.Context) {
	now := time.Now()
	nextMinute := now.Truncate(time.Minute).Add(time.Minute)
	time.Sleep(nextMinute.Sub(now))

	ticker := time.NewTicker(time.Minute)
	defer ticker.Stop()

	log.Info("Starting mailing service")

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			users, err := s.userRepo.GetAllUsers()
			if err != nil {
				log.Errorf("Error getting users: %v", err)
				continue
			}

			now := time.Now()
			for _, user := range users {
				timezone, err := strconv.Atoi(user.Timezone)
				if err != nil {
					log.Errorf("Error converting timezone to int: %v", err)
					continue
				}
				userLoc := time.FixedZone(user.Timezone, timezone*60*60)
				nowInUserZone := now.In(userLoc)

				if nowInUserZone.Hour() == user.MailingTime.Hour() &&
					nowInUserZone.Minute() == user.MailingTime.Minute() {
					log.Infof("Sending mailing to user %d", user.ChatID)
					s.mailingChan <- user
				}
			}
		}
	}
}

func (s *MailingService) sendMailings(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			return
		case user := <-s.mailingChan:
			// Get weather
			weatherMsg, err := s.getWeatherMessage(user.City)
			if err != nil {
				log.Errorf("Error getting weather: %v", err)
				continue
			}

			ratesMsg, err := s.getRatesMessage()
			if err != nil {
				log.Errorf("Error getting rates: %v", err)
				continue
			}

			newsMsg, err := s.getNewsMessage(user.PreferredChannelID)
			if err != nil {
				log.Errorf("Error getting news: %v", err)
				continue
			}

			fullMessage := fmt.Sprintf("Ежедневная рассылка:\n\n%s\n\n%s\n\n%s",
				weatherMsg, ratesMsg, newsMsg)

			_, err = s.bot.Send(&tele.User{ID: user.ChatID}, fullMessage, keyboard.GetStartKeyboard())
			if err != nil {
				log.Errorf("Error sending message to user %d: %v", user.ChatID, err)
			}
		}
	}
}

func (s *MailingService) getWeatherMessage(city string) (string, error) {
	resp, err := s.weatherRepo.GetWeatherByCity(city)
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("Погода в городе: %s\n%.1f градусов\n%s",
		resp.City, resp.Main.Temp, resp.Weather[0].Desc), nil
}

func (s *MailingService) getRatesMessage() (string, error) {
	rates, err := s.rateRepo.GetRates()
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("**Курс валют на сегодня**\n"+
		"Доллар: %.2f ₽ (изменение: %.2f%%)\n"+
		"Евро: %.2f ₽ (изменение: %.2f%%)",
		rates.USD.Value,
		((rates.USD.Value-rates.USD.Previous)/rates.USD.Previous)*100,
		rates.EUR.Value,
		((rates.EUR.Value-rates.EUR.Previous)/rates.EUR.Previous)*100), nil
}

func (s *MailingService) getNewsMessage(channelID int64) (string, error) {
	summary, err := s.summaryRepo.GetLatestSummary(channelID)
	if err != nil {
		return "", err
	}
	if summary == nil {
		return "Новостей пока нет. Проверьте позже.", nil
	}
	return fmt.Sprintf("Последние новости:\n%s", summary.Summary), nil
}
