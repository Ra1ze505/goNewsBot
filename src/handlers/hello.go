package handlers

import (
	"fmt"

	"github.com/Ra1ze505/goNewsBot/src/keyboard"
	tele "gopkg.in/telebot.v4"
)

func HelloHandle(context tele.Context) error {
	return context.Send(fmt.Sprintf("Привет, @%s", context.Sender().Username), keyboard.GetStartKeyboard())
}
