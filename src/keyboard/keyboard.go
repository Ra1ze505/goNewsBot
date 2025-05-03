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

	ChangePrimeChannelBtn = tele.Btn{Text: "Изменить канал"}

	Time8Btn  = tele.Btn{Text: "08:00"}
	Time9Btn  = tele.Btn{Text: "09:00"}
	Time10Btn = tele.Btn{Text: "10:00"}

	CancelBtn = tele.Btn{
		Text: "Отмена",
		Data: "cancel_channel",
	}
)

func GetStartKeyboard() *tele.ReplyMarkup {
	keyboard := &tele.ReplyMarkup{
		ResizeKeyboard: true,
	}

	keyboard.Reply(
		tele.Row{WeatherBtn, RateBtn, NewsBtn},
		tele.Row{ChangePrimeChannelBtn, ChangeCityBtn},
		tele.Row{ChangeTimeBtn},
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

func GetTimeSelectionKeyboard() *tele.ReplyMarkup {
	keyboard := &tele.ReplyMarkup{
		ResizeKeyboard: true,
	}

	keyboard.Reply(
		tele.Row{Time8Btn, Time9Btn, Time10Btn},
		tele.Row{CancelBtn},
	)

	return keyboard
}
