package handlers

import (
	"fmt"

	"github.com/Ra1ze505/goNewsBot/src/keyboard"
	"github.com/Ra1ze505/goNewsBot/src/repository"
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
	if !isValidSummaryLength(message) {
		return c.Send("Суммарная длина новостей превышает 4096 символов. Воспользуйтесь кнопкой 'Написать нам' и сообщите о проблеме.", keyboard.GetStartKeyboard())
	}

	err = c.Send(message, keyboard.GetStartKeyboard(), &tele.SendOptions{
		ParseMode: tele.ModeMarkdown,
	})

	if err != nil {
		log.Errorf("Error sending news message to user %d: %v", c.Sender().ID, err)
		log.Info("Try send plain text message")
		err = c.Send(message, keyboard.GetStartKeyboard())
		if err != nil {
			log.Errorf("Error sending plain text message to user %d: %v", c.Sender().ID, err)
			return err
		}
	}
	return nil

}

func isValidSummaryLength(summary string) bool {
	return len(summary) <= 4096
}
