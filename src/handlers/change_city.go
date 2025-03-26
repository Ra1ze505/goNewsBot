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
	user, ok := c.Get("user").(*repository.User)
	if !ok {
		return fmt.Errorf("user not found in context")
	}

	h.stateStorage.SetState(user.ChatID, &UserState{ChangingCity: true})

	return c.Send(fmt.Sprintf("Ваш город сейчас: %s\nВыберите город из списка или напишите свой", user.City), keyboard.GetCitySelectionKeyboard())
}

func (h *ChangeCityHandler) HandleCityInput(c tele.Context) error {
	user, ok := c.Get("user").(*repository.User)
	if !ok {
		return fmt.Errorf("user not found in context")
	}

	state := h.stateStorage.GetState(user.ChatID)
	if state == nil || !state.ChangingCity {
		return nil
	}

	if c.Text() == keyboard.CancelCityBtn.Text {
		h.stateStorage.ClearState(user.ChatID)
		return c.Send("Город не изменен", keyboard.GetStartKeyboard())
	}

	weather, err := h.weatherRepo.GetWeatherByCity(c.Text())
	if err != nil {
		return c.Send("Некорректный город\nПопробуйте еще раз", keyboard.GetStartKeyboard())
	}

	timezone := strconv.Itoa(weather.Timezone / 3600)

	log.Infof("Upating user.id: %d, city: %s, timezone: %s", user.ID, weather.City, timezone)
	err = h.userRepo.UpdateUserCityAndTimezone(user.ID, weather.City, timezone)
	if err != nil {
		return fmt.Errorf("failed to update user: %w", err)
	}

	h.stateStorage.ClearState(user.ChatID)

	return c.Send(fmt.Sprintf("Город изменен на %s", weather.City), keyboard.GetStartKeyboard())
}
