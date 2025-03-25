package handlers

import (
	"github.com/Ra1ze505/goNewsBot/src/keyboard"
	tele "gopkg.in/telebot.v4"
)

func AboutHandle(c tele.Context) error {
	return c.Send("Это информационный бот, который предоставляет:\n- Погоду\n- Курсы валют\n- Новости\n\nВыберите нужный раздел в меню ниже.", keyboard.GetStartKeyboard())
}
