package handlers

import (
	tele "gopkg.in/telebot.v4"
)

func EchoHandle(context tele.Context) error {
	text := context.Text()
	return context.Send(text)

}
