package main

import (
	"os"
	"time"

	handlers "github.com/Ra1ze505/goNewsBot/src/handlers"
	middleware "github.com/Ra1ze505/goNewsBot/src/middleware"

	"github.com/joho/godotenv"
	log "github.com/sirupsen/logrus"

	tele "gopkg.in/telebot.v4"
)

func loadEnv() {
	err := godotenv.Load()
	if err != nil {
		log.Fatal("Error loading .env file")
	}

}

func main() {
	log.Info("Start ...")
	loadEnv()
	pref := tele.Settings{
		Token:  os.Getenv("BOT_TOKEN"),
		Poller: &tele.LongPoller{Timeout: 10 * time.Second},
	}

	bot, err := tele.NewBot(pref)
	if err != nil {
		log.Fatal(err)
		return
	}

	bot.Use(middleware.MessageLogger())

	addHandlers(bot)
	bot.Start()
}

func addHandlers(bot *tele.Bot) {

	bot.Handle("/hello", handlers.HelloHandle)
	bot.Handle("/weather", handlers.WeatherHandle)
	bot.Handle(tele.OnText, handlers.EchoHandle)
}
