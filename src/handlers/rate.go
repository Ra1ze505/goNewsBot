package handlers

import (
	"github.com/Ra1ze505/goNewsBot/src/keyboard"
	tele "gopkg.in/telebot.v4"
)

func RateHandle(c tele.Context) error {
	return c.Send("Текущий курс: 1 USD = 90 RUB", keyboard.GetStartKeyboard())
}
