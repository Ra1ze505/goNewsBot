package handlers

import (
	"fmt"

	"github.com/Ra1ze505/goNewsBot/src/keyboard"
	"github.com/Ra1ze505/goNewsBot/src/repository"
	"github.com/Ra1ze505/goNewsBot/src/telegramutil"
	tele "gopkg.in/telebot.v4"

	log "github.com/sirupsen/logrus"
)

type NewsHandler struct {
	summaryRepo repository.SummaryRepositoryInterface
}

func NewNewsHandler(summaryRepo repository.SummaryRepositoryInterface) *NewsHandler {
	return &NewsHandler{summaryRepo: summaryRepo}
}

func (h *NewsHandler) Handle(c tele.Context) error {
	user, ok := c.Get("user").(*repository.User)
	if !ok {
		return fmt.Errorf("user not found in context")
	}

	summary, err := h.summaryRepo.GetLatestSummary(user.PreferredChannelID)
	if err != nil {
		log.Errorf("Error getting latest summary: %v", err)
		return c.Send("Произошла ошибка при получении новостей. Попробуйте позже.", keyboard.GetStartKeyboard())
	}

	if summary == nil {
		return c.Send("Новостей пока нет. Проверьте позже.", keyboard.GetStartKeyboard())
	}

	message := summary.GetFormattedSummary()
	parts := telegramutil.SplitMessage(message)

	for i, part := range parts {
		opts := makeSendOptions(i == len(parts)-1)

		err = c.Send(part, opts...)
		if err != nil {
			log.Errorf("Error sending news message part %d/%d to user %d: %v", i+1, len(parts), c.Sender().ID, err)
			log.Info("Try send plain text message")
			err = c.Send(part, makePlainSendOptions(i == len(parts)-1)...)
			if err != nil {
				log.Errorf("Error sending plain text message part %d/%d to user %d: %v", i+1, len(parts), c.Sender().ID, err)
				return err
			}
		}
	}

	return nil
}

func makeSendOptions(withKeyboard bool) []interface{} {
	opts := []interface{}{
		&tele.SendOptions{ParseMode: tele.ModeMarkdown},
	}
	if withKeyboard {
		opts = append([]interface{}{keyboard.GetStartKeyboard()}, opts...)
	}
	return opts
}

func makePlainSendOptions(withKeyboard bool) []interface{} {
	if withKeyboard {
		return []interface{}{keyboard.GetStartKeyboard()}
	}
	return nil
}
