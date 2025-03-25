package keyboard

import tele "gopkg.in/telebot.v4"

// Button objects
var (
	WeatherBtn    = tele.Btn{Text: "Погода"}
	RateBtn       = tele.Btn{Text: "Курс"}
	NewsBtn       = tele.Btn{Text: "Новости"}
	ChangeCityBtn = tele.Btn{Text: "Изменить город"}
	ChangeTimeBtn = tele.Btn{Text: "Изменить время рассылки"}
	AboutBtn      = tele.Btn{Text: "О боте"}
	ContactBtn    = tele.Btn{Text: "Написать нам"}
)

// GetStartKeyboard returns the main menu keyboard layout
func GetStartKeyboard() *tele.ReplyMarkup {
	keyboard := &tele.ReplyMarkup{
		ResizeKeyboard: true,
	}

	// Create rows
	keyboard.Reply(
		tele.Row{WeatherBtn, RateBtn, NewsBtn},
		tele.Row{ChangeCityBtn, ChangeTimeBtn},
		tele.Row{AboutBtn, ContactBtn},
	)

	return keyboard
}
