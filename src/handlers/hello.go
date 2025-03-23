package handlers

import (
	tele "gopkg.in/telebot.v4"
)

func HelloHandle(context tele.Context) error {
	return context.Send("Hi!")
}
