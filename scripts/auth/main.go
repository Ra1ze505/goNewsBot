package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"path/filepath"
	"strconv"

	"github.com/go-faster/errors"
	"github.com/gotd/contrib/middleware/floodwait"
	"github.com/gotd/td/examples"
	"github.com/gotd/td/telegram"
	"github.com/gotd/td/telegram/auth"
	"github.com/joho/godotenv"
	"go.uber.org/zap"

	"github.com/Ra1ze505/goNewsBot/src/config"
)

func loadEnv() (int, string, error) {
	if err := godotenv.Load(); err != nil && !os.IsNotExist(err) {
		return 0, "", errors.Wrap(err, "load env")
	}

	appID, err := strconv.Atoi(os.Getenv("API_ID"))
	if err != nil {
		return 0, "", errors.Wrap(err, "parse app id")
	}

	appHash := os.Getenv("API_HASH")
	if appHash == "" {
		return 0, "", errors.New("no app hash")
	}

	return appID, appHash, nil
}

func initClient(appID int, appHash string) (*telegram.Client, *floodwait.Waiter, error) {
	sessionDir := config.SessionDir
	if err := os.MkdirAll(sessionDir, 0700); err != nil {
		return nil, nil, err
	}

	waiter := floodwait.NewWaiter().WithCallback(func(ctx context.Context, wait floodwait.FloodWait) {
		fmt.Println("Got FLOOD_WAIT. Will retry after", wait.Duration)
	})

	options := telegram.Options{
		Logger: zap.NewNop(),
		SessionStorage: &telegram.FileSessionStorage{
			Path: filepath.Join(sessionDir, "session.json"),
		},
		Middlewares: []telegram.Middleware{
			waiter,
		},
	}

	client := telegram.NewClient(appID, appHash, options)
	return client, waiter, nil
}

func run(ctx context.Context) error {
	phone := flag.String("phone", "", "Phone number for authentication")
	flag.Parse()

	if *phone == "" {
		return errors.New("phone number is required, use --phone flag")
	}

	appID, appHash, err := loadEnv()
	if err != nil {
		return err
	}

	client, waiter, err := initClient(appID, appHash)
	if err != nil {
		return err
	}

	flow := auth.NewFlow(examples.Terminal{PhoneNumber: *phone}, auth.SendCodeOptions{})

	return waiter.Run(ctx, func(ctx context.Context) error {
		if err := client.Run(ctx, func(ctx context.Context) error {
			if err := client.Auth().IfNecessary(ctx, flow); err != nil {
				return errors.Wrap(err, "auth")
			}
			fmt.Println("Successfully authenticated and saved session")
			return nil
		}); err != nil {
			return errors.Wrap(err, "run")
		}
		return nil
	})
}

func main() {
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt)
	defer cancel()

	if err := run(ctx); err != nil {
		if errors.Is(err, context.Canceled) && ctx.Err() == context.Canceled {
			fmt.Println("\rClosed")
			os.Exit(0)
		}
		_, _ = fmt.Fprintf(os.Stderr, "Error: %+v\n", err)
		os.Exit(1)
	} else {
		fmt.Println("Done")
		os.Exit(0)
	}
}
