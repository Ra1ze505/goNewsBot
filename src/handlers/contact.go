package handlers

import (
	"github.com/Ra1ze505/goNewsBot/src/keyboard"
	tele "gopkg.in/telebot.v4"
)

func ContactHandle(c tele.Context) error {
	return c.Send("По всем вопросам обращайтесь к @ra1zeee", keyboard.GetStartKeyboard())
}
