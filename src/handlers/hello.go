package handlers

import (
	"fmt"

	tele "gopkg.in/telebot.v4"
)

func HelloHandle(context tele.Context) error {
	context.Send(fmt.Sprintf("Привет, @%s", context.Sender().Username))

	return nil
}

