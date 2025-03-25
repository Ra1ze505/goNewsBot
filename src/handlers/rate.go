package handlers

import (
	"fmt"

	"github.com/Ra1ze505/goNewsBot/src/keyboard"
	"github.com/Ra1ze505/goNewsBot/src/repository"
	tele "gopkg.in/telebot.v4"
)

type RateHandler struct {
	rateRepo repository.RateRepositoryInterface
}

func NewRateHandler(rateRepo repository.RateRepositoryInterface) *RateHandler {
	return &RateHandler{rateRepo: rateRepo}
}

func (h *RateHandler) Handle(c tele.Context) error {
	rates, err := h.rateRepo.GetRates()
	if err != nil || rates == nil {
		return c.Send("Извините, не удалось получить текущий курс валют. Попробуйте позже.", keyboard.GetStartKeyboard())
	}

	message := fmt.Sprintf("**Курс валют на сегодня**\n"+
		"Доллар: %.2f ₽ (изменение: %.2f%%)\n"+
		"Евро: %.2f ₽ (изменение: %.2f%%)",
		rates.USD.Value,
		((rates.USD.Value-rates.USD.Previous)/rates.USD.Previous)*100,
		rates.EUR.Value,
		((rates.EUR.Value-rates.EUR.Previous)/rates.EUR.Previous)*100)

	return c.Send(message, keyboard.GetStartKeyboard())
}
