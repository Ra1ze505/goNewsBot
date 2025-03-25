package handlers

import (
	"github.com/Ra1ze505/goNewsBot/src/keyboard"
	tele "gopkg.in/telebot.v4"
)

func ChangeTimeHandle(c tele.Context) error {
	return c.Send("Введите время для рассылки в формате ЧЧ:ММ (например, 09:00):", keyboard.GetStartKeyboard())
}
