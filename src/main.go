package main

import (
	"database/sql"
	"os"
	"time"

	"github.com/Ra1ze505/goNewsBot/src/handlers"
	"github.com/Ra1ze505/goNewsBot/src/middleware"
	"github.com/Ra1ze505/goNewsBot/src/repository"

	"github.com/joho/godotenv"
	_ "github.com/lib/pq"
	"github.com/pkg/errors"
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
		log.Error(err)
		return
	}

	db, err := sql.Open("postgres", os.Getenv("DATABASE_URL"))
	if err != nil {
		log.Fatal(errors.Wrap(err, "failed to open database connection"))
	}
	defer db.Close()

	userRepo := repository.NewUserRepository(db)

	bot.Use(middleware.MessageLogger())
	bot.Use(middleware.CreateOrUpdateUser(userRepo))
	addHandlers(bot)
	bot.Start()
}

func addHandlers(bot *tele.Bot) {
	bot.Handle("/start", handlers.HelloHandle)
	bot.Handle("/weather", handlers.WeatherHandle)
	bot.Handle(tele.OnText, handlers.EchoHandle)
}
