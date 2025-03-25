package handlers

import (
	"github.com/Ra1ze505/goNewsBot/src/keyboard"
	tele "gopkg.in/telebot.v4"
)

func NewsHandle(c tele.Context) error {
	return c.Send("Последние новости:\n1. Важная новость 1\n2. Важная новость 2", keyboard.GetStartKeyboard())
}
