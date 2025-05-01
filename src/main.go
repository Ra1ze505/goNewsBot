package main

import (
	"context"
	"database/sql"
	"os"
	"time"

	"github.com/Ra1ze505/goNewsBot/src/handlers"
	"github.com/Ra1ze505/goNewsBot/src/keyboard"
	"github.com/Ra1ze505/goNewsBot/src/middleware"
	"github.com/Ra1ze505/goNewsBot/src/repository"
	"github.com/Ra1ze505/goNewsBot/src/service"

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
	weatherRepo := repository.NewWeatherRepository()
	stateStorage := handlers.NewStateStorage()

	rateRepo := repository.NewRateRepository(db)
	rateService := service.NewRateService(rateRepo)
	rateService.StartRateFetcher()

	ctx := context.Background()
	messageService, err := service.InitAndStartMessageService(ctx, db)
	if err != nil {
		log.Fatal(errors.Wrap(err, "Failed to initialize message service"))
	}

	mlRepo, err := repository.NewMLRepository()
	if err != nil {
		log.Fatal(errors.Wrap(err, "Failed to initialize ML repository"))
	}

	summaryRepo := repository.NewSummaryRepository(db)
	summaryService := service.NewSummaryService(summaryRepo, mlRepo, messageService.MessagesFetched)
	summaryService.StartSummaryFetcher(ctx)

	log.Info("Bot started")
	bot.Use(middleware.MessageLogger())
	bot.Use(middleware.CreateOrUpdateUser(userRepo))
	addHandlers(bot, userRepo, weatherRepo, stateStorage, rateRepo)
	bot.Start()
}

func addHandlers(bot *tele.Bot, userRepo repository.UserRepositoryInterface, weatherRepo repository.WeatherRepositoryInterface, stateStorage *handlers.StateStorage, rateRepo repository.RateRepositoryInterface) {
	// Start command
	bot.Handle("/start", handlers.HelloHandle)

	// Initialize handlers
	changeCityHandler := handlers.NewChangeCityHandler(userRepo, weatherRepo, stateStorage)
	rateHandler := handlers.NewRateHandler(rateRepo)

	// Button handlers
	bot.Handle(&keyboard.WeatherBtn, handlers.WeatherHandle)
	bot.Handle(&keyboard.RateBtn, rateHandler.Handle)
	bot.Handle(&keyboard.NewsBtn, handlers.NewsHandle)
	bot.Handle(&keyboard.ChangeCityBtn, changeCityHandler.Handle)
	bot.Handle(&keyboard.ChangeTimeBtn, handlers.ChangeTimeHandle)
	bot.Handle(&keyboard.AboutBtn, handlers.AboutHandle)
	bot.Handle(&keyboard.ContactBtn, handlers.ContactHandle)

	// Text message handler
	bot.Handle(tele.OnText, func(c tele.Context) error {
		user, ok := c.Get("user").(*repository.User)
		if !ok {
			return nil
		}

		state := stateStorage.GetState(user.ChatID)
		if state != nil && state.ChangingCity {
			return changeCityHandler.HandleCityInput(c)
		}
		return nil
	})
}
