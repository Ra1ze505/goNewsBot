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

	ChangePrimeChannelBtn = tele.Btn{
		Text: "Изменить новостной канал",
		Data: "change_prime_channel",
	}

	CancelChannelBtn = tele.Btn{
		Text: "Отмена",
		Data: "cancel_channel",
	}
)

func GetStartKeyboard() *tele.ReplyMarkup {
	keyboard := &tele.ReplyMarkup{
		ResizeKeyboard: true,
	}

	keyboard.Reply(
		keyboard.Row(WeatherBtn),
		keyboard.Row(RateBtn),
		keyboard.Row(NewsBtn),
		keyboard.Row(ChangeCityBtn),
		keyboard.Row(ChangeTimeBtn),
		keyboard.Row(ChangePrimeChannelBtn),
		keyboard.Row(AboutBtn),
		keyboard.Row(ContactBtn),
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
