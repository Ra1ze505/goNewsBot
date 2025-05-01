package handlers

import (
	"fmt"

	"github.com/Ra1ze505/goNewsBot/src/keyboard"
	"github.com/Ra1ze505/goNewsBot/src/repository"
	tele "gopkg.in/telebot.v4"
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
		return c.Send("Произошла ошибка при получении новостей. Попробуйте позже.", keyboard.GetStartKeyboard())
	}

	if summary == nil {
		return c.Send("Новостей пока нет. Проверьте позже.", keyboard.GetStartKeyboard())
	}

	message := fmt.Sprintf("Последние новости:\n\n%s", summary.Summary)
	return c.Send(message, keyboard.GetStartKeyboard())
}
