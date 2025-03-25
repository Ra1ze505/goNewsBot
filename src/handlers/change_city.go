package handlers

import (
	"fmt"
	"strconv"

	"github.com/Ra1ze505/goNewsBot/src/keyboard"
	"github.com/Ra1ze505/goNewsBot/src/repository"
	log "github.com/sirupsen/logrus"
	tele "gopkg.in/telebot.v4"
)

type ChangeCityHandler struct {
	userRepo     repository.UserRepositoryInterface
	weatherRepo  repository.WeatherRepositoryInterface
	stateStorage *StateStorage
}

func NewChangeCityHandler(userRepo repository.UserRepositoryInterface, weatherRepo repository.WeatherRepositoryInterface, stateStorage *StateStorage) *ChangeCityHandler {
	return &ChangeCityHandler{
		userRepo:     userRepo,
		weatherRepo:  weatherRepo,
		stateStorage: stateStorage,
	}
}

func (h *ChangeCityHandler) Handle(c tele.Context) error {
	// Get user from context
	user, ok := c.Get("user").(*repository.User)
	if !ok {
		return fmt.Errorf("user not found in context")
	}

	// Set state for user
	h.stateStorage.SetState(user.ChatID, &UserState{ChangingCity: true})

	// Send initial message
	return c.Send(fmt.Sprintf("Ваш город сейчас: %s\nНапишите свой город", user.City), keyboard.GetStartKeyboard())
}

func (h *ChangeCityHandler) HandleCityInput(c tele.Context) error {
	// Get user from context
	user, ok := c.Get("user").(*repository.User)
	if !ok {
		return fmt.Errorf("user not found in context")
	}

	// Get user state
	state := h.stateStorage.GetState(user.ChatID)
	if state == nil || !state.ChangingCity {
		return nil
	}

	if c.Text() == "Отмена" {
		h.stateStorage.ClearState(user.ChatID)
		return c.Send("Город не изменен", keyboard.GetStartKeyboard())
	}

	// Try to get weather for the new city
	weather, err := h.weatherRepo.GetWeatherByCity(c.Text())
	if err != nil {
		return c.Send("Некорректный город\nПопробуйте еще раз", keyboard.GetStartKeyboard())
	}

	// Convert timezone offset to string
	timezone := strconv.Itoa(weather.Timezone / 3600)

	// Update user's city and timezone
	log.Infof("Upating user.id: %d, city: %s, timezone: %s", user.ID, weather.City, timezone)
	err = h.userRepo.UpdateUserCityAndTimezone(user.ID, weather.City, timezone)
	if err != nil {
		return fmt.Errorf("failed to update user: %w", err)
	}

	// Clear state after successful update
	h.stateStorage.ClearState(user.ChatID)

	return c.Send(fmt.Sprintf("Город изменен на %s", weather.City), keyboard.GetStartKeyboard())
}
