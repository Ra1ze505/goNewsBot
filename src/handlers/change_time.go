package handlers

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/Ra1ze505/goNewsBot/src/keyboard"
	"github.com/Ra1ze505/goNewsBot/src/repository"
	log "github.com/sirupsen/logrus"
	tele "gopkg.in/telebot.v4"
)

type ChangeTimeHandler struct {
	userRepo     repository.UserRepositoryInterface
	stateStorage *StateStorage
}

func NewChangeTimeHandler(userRepo repository.UserRepositoryInterface, stateStorage *StateStorage) *ChangeTimeHandler {
	return &ChangeTimeHandler{
		userRepo:     userRepo,
		stateStorage: stateStorage,
	}
}

func (h *ChangeTimeHandler) Handle(c tele.Context) error {
	user, ok := c.Get("user").(*repository.User)
	if !ok {
		return fmt.Errorf("user not found in context")
	}

	h.stateStorage.SetState(user.ChatID, &UserState{ChangingTime: true})

	timeStr := user.MailingTime.Format("15:04")

	message := fmt.Sprintf("Ваше текущее время рассылки: %s\n\nВыберите время из предложенных или введите свое в формате ЧЧ:ММ (например, 09:00):", timeStr)

	return c.Send(message, keyboard.GetTimeSelectionKeyboard())
}

func (h *ChangeTimeHandler) HandleTimeInput(c tele.Context) error {
	user, ok := c.Get("user").(*repository.User)
	if !ok {
		return fmt.Errorf("user not found in context")
	}

	state := h.stateStorage.GetState(user.ChatID)
	if state == nil || !state.ChangingTime {
		return nil
	}

	if c.Text() == keyboard.CancelBtn.Text {
		h.stateStorage.ClearState(user.ChatID)
		return c.Send("Время не изменено", keyboard.GetStartKeyboard())
	}

	parts := strings.Split(c.Text(), ":")
	if len(parts) != 2 {
		return c.Send("Некорректный формат времени. Используйте формат ЧЧ:ММ (например, 09:00)", keyboard.GetStartKeyboard())
	}

	hour, err := strconv.Atoi(parts[0])
	if err != nil || hour < 0 || hour > 23 {
		return c.Send("Некорректный час. Используйте значения от 00 до 23", keyboard.GetStartKeyboard())
	}

	minute, err := strconv.Atoi(parts[1])
	if err != nil || minute < 0 || minute > 59 {
		return c.Send("Некорректные минуты. Используйте значения от 00 до 59", keyboard.GetStartKeyboard())
	}

	localTime := time.Date(0, 0, 0, hour, minute, 0, 0, time.Local)

	log.Infof("Updating user.id: %d, mailing time: %s", user.ID, localTime.Format("15:04"))
	err = h.userRepo.UpdateUserMailingTime(user.ID, localTime)
	if err != nil {
		return fmt.Errorf("failed to update user mailing time: %w", err)
	}

	h.stateStorage.ClearState(user.ChatID)

	return c.Send(fmt.Sprintf("Время рассылки изменено на %s", c.Text()), keyboard.GetStartKeyboard())
}
