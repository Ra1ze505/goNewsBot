package keyboard

import tele "gopkg.in/telebot.v4"

var (
	WeatherBtn    = tele.Btn{Text: "Погода"}
	RateBtn       = tele.Btn{Text: "Курс"}
	NewsBtn       = tele.Btn{Text: "Новости"}
	ChangeCityBtn = tele.Btn{Text: "Изменить город"}
	ChangeTimeBtn = tele.Btn{Text: "Изменить время рассылки"}
	AboutBtn      = tele.Btn{Text: "О боте"}
	ContactBtn    = tele.Btn{Text: "Написать нам"}

	// City selection buttons
	MoscowBtn     = tele.Btn{Text: "Москва"}
	StPetersBtn   = tele.Btn{Text: "Санкт-Петербург"}
	CancelCityBtn = tele.Btn{Text: "Отмена"}
)

func GetStartKeyboard() *tele.ReplyMarkup {
	keyboard := &tele.ReplyMarkup{
		ResizeKeyboard: true,
	}

	keyboard.Reply(
		tele.Row{WeatherBtn, RateBtn, NewsBtn},
		tele.Row{ChangeCityBtn, ChangeTimeBtn},
		tele.Row{AboutBtn, ContactBtn},
	)

	return keyboard
}

func GetCitySelectionKeyboard() *tele.ReplyMarkup {
	keyboard := &tele.ReplyMarkup{
		ResizeKeyboard: true,
	}

	keyboard.Reply(
		tele.Row{MoscowBtn, StPetersBtn},
		tele.Row{CancelCityBtn},
	)

	return keyboard
}
