package handlers

import (
	"fmt"

	"github.com/Ra1ze505/goNewsBot/src/config"
	"github.com/Ra1ze505/goNewsBot/src/keyboard"
	"github.com/Ra1ze505/goNewsBot/src/repository"
	tele "gopkg.in/telebot.v4"
)

type ChangePrimeChannelHandler struct {
	userRepo repository.UserRepositoryInterface
}

func NewChangePrimeChannelHandler(userRepo repository.UserRepositoryInterface) *ChangePrimeChannelHandler {
	return &ChangePrimeChannelHandler{
		userRepo: userRepo,
	}
}

func (h *ChangePrimeChannelHandler) Handle(c tele.Context) error {
	user, ok := c.Get("user").(*repository.User)
	if !ok {
		return fmt.Errorf("user not found in context")
	}

	// Get current channel name
	currentChannelName := config.Channels[user.PreferredChannelID]
	if currentChannelName == "" {
		currentChannelName = "неизвестный канал"
	}

	// Create message with current channel and available options
	message := fmt.Sprintf("Ваш текущий новостной канал: %s\n\nВыберите новый канал:", currentChannelName)

	// Create keyboard with available channels
	var rows []tele.Row
	for channelID, channelName := range config.Channels {
		btn := tele.Btn{
			Text: channelName,
			Data: fmt.Sprintf("channel_%d", channelID),
		}
		rows = append(rows, tele.Row{btn})
	}

	rows = append(rows, tele.Row{keyboard.CancelChannelBtn})

	keyboard := &tele.ReplyMarkup{
		ResizeKeyboard: true,
	}
	keyboard.Inline(rows...)

	return c.Send(message, keyboard)
}

func (h *ChangePrimeChannelHandler) HandleChannelSelection(c tele.Context) error {
	user, ok := c.Get("user").(*repository.User)
	if !ok {
		return fmt.Errorf("user not found in context")
	}

	if c.Callback().Data == keyboard.CancelChannelBtn.Data {
		return c.Send("Выбор канала отменен", keyboard.GetStartKeyboard())
	}

	var channelID int64
	_, err := fmt.Sscanf(c.Callback().Data, "channel_%d", &channelID)
	if err != nil {
		return fmt.Errorf("failed to parse channel ID: %w", err)
	}

	err = h.userRepo.UpdatePreferredChannel(user.ID, channelID)
	if err != nil {
		return fmt.Errorf("failed to update preferred channel: %w", err)
	}

	channelName := config.Channels[channelID]
	if channelName == "" {
		channelName = "неизвестный канал"
	}

	return c.Send(fmt.Sprintf("Новостной канал изменен на: %s", channelName), keyboard.GetStartKeyboard())
}
