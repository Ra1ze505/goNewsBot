package main

import (
	"context"
	"database/sql"
	"os"
	"strings"
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

type Repositories struct {
	UserRepository    repository.UserRepositoryInterface
	RateRepository    repository.RateRepositoryInterface
	SummaryRepository repository.SummaryRepositoryInterface
	MessageRepository repository.MessageRepositoryInterface
	MLRepository      repository.MLRepositoryInterface
	WeatherRepository repository.WeatherRepositoryInterface
	StateStorage      *handlers.StateStorage
}

func NewRepositories(db *sql.DB) *Repositories {
	mlRepo, err := repository.NewMLRepository()
	if err != nil {
		log.Fatal(errors.Wrap(err, "Failed to initialize ML repository"))
	}
	return &Repositories{
		UserRepository:    repository.NewUserRepository(db),
		RateRepository:    repository.NewRateRepository(db),
		SummaryRepository: repository.NewSummaryRepository(db),
		MessageRepository: repository.NewMessageRepository(db),
		MLRepository:      mlRepo,
		WeatherRepository: repository.NewWeatherRepository(),
		StateStorage:      handlers.NewStateStorage(),
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

	repositories := NewRepositories(db)

	rateService := service.NewRateService(repositories.RateRepository)
	rateService.StartRateFetcher()

	ctx := context.Background()
	messageService, err := service.InitAndStartMessageService(ctx, db)
	if err != nil {
		log.Fatal(errors.Wrap(err, "Failed to initialize message service"))
	}

	summaryService := service.NewSummaryService(repositories.SummaryRepository, repositories.MLRepository, messageService.MessagesFetched)
	summaryService.StartSummaryFetcher(ctx)

	bot.Use(middleware.MessageLogger())
	bot.Use(middleware.CreateOrUpdateUser(repositories.UserRepository))
	addHandlers(bot, repositories)
	bot.Start()
}

func addHandlers(bot *tele.Bot, repositories *Repositories) {
	// Start command
	bot.Handle("/start", handlers.HelloHandle)

	// Initialize handlers
	changeCityHandler := handlers.NewChangeCityHandler(repositories.UserRepository, repositories.WeatherRepository, repositories.StateStorage)
	rateHandler := handlers.NewRateHandler(repositories.RateRepository)
	newsHandler := handlers.NewNewsHandler(repositories.SummaryRepository)
	changePrimeChannelHandler := handlers.NewChangePrimeChannelHandler(repositories.UserRepository)

	// Button handlers
	bot.Handle(&keyboard.WeatherBtn, handlers.WeatherHandle)
	bot.Handle(&keyboard.RateBtn, rateHandler.Handle)
	bot.Handle(&keyboard.NewsBtn, newsHandler.Handle)
	bot.Handle(&keyboard.ChangeCityBtn, changeCityHandler.Handle)
	bot.Handle(&keyboard.ChangeTimeBtn, handlers.ChangeTimeHandle)
	bot.Handle(&keyboard.AboutBtn, handlers.AboutHandle)
	bot.Handle(&keyboard.ContactBtn, handlers.ContactHandle)
	bot.Handle(&keyboard.ChangePrimeChannelBtn, changePrimeChannelHandler.Handle)

	// Handle callback queries
	bot.Handle(tele.OnCallback, func(c tele.Context) error {
		if c.Callback().Data == keyboard.CancelChannelBtn.Data || strings.HasPrefix(c.Callback().Data, "channel_") {
			return changePrimeChannelHandler.HandleChannelSelection(c)
		}
		return nil
	})

	// Text message handler
	bot.Handle(tele.OnText, func(c tele.Context) error {
		user, ok := c.Get("user").(*repository.User)
		if !ok {
			return nil
		}

		state := repositories.StateStorage.GetState(user.ChatID)
		if state != nil && state.ChangingCity {
			return changeCityHandler.HandleCityInput(c)
		}
		return nil
	})
}
