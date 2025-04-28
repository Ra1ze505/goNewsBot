package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"path/filepath"
	"strconv"
	"time"

	"github.com/go-faster/errors"
	"github.com/gotd/contrib/middleware/floodwait"
	"github.com/gotd/td/examples"
	"github.com/gotd/td/telegram"
	"github.com/gotd/td/telegram/auth"
	"github.com/gotd/td/tg"
	"github.com/joho/godotenv"
	"go.uber.org/zap"
)

func sessionFolder(phone string) string {
	var out []rune
	for _, r := range phone {
		if r >= '0' && r <= '9' {
			out = append(out, r)
		}
	}
	return "phone-" + string(out)
}

// parseFlags parses command line flags
func parseFlags() (string, error) {
	var channelName string
	flag.StringVar(&channelName, "channel", "", "Channel name to get messages from")
	flag.Parse()

	if channelName == "" {
		return "", errors.New("channel name is required")
	}

	return channelName, nil
}

// loadEnv loads environment variables from .env file
func loadEnv() (string, int, string, error) {
	if err := godotenv.Load(); err != nil && !os.IsNotExist(err) {
		return "", 0, "", errors.Wrap(err, "load env")
	}

	phone := os.Getenv("TG_PHONE")
	if phone == "" {
		return "", 0, "", errors.New("no phone")
	}

	appID, err := strconv.Atoi(os.Getenv("API_ID"))
	if err != nil {
		return "", 0, "", errors.Wrap(err, "parse app id")
	}

	appHash := os.Getenv("API_HASH")
	if appHash == "" {
		return "", 0, "", errors.New("no app hash")
	}

	return phone, appID, appHash, nil
}

// initClient initializes Telegram client with the given parameters
func initClient(phone string, appID int, appHash string) (*telegram.Client, *floodwait.Waiter, error) {
	sessionDir := filepath.Join("session", sessionFolder(phone))
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

// getChannel resolves a channel by its username
func getChannel(ctx context.Context, api *tg.Client, username string) (*tg.Channel, error) {
	resolved, err := api.ContactsResolveUsername(ctx, &tg.ContactsResolveUsernameRequest{
		Username: username,
	})
	if err != nil {
		return nil, errors.Wrap(err, "resolve username")
	}

	channel, ok := resolved.Chats[0].(*tg.Channel)
	if !ok {
		return nil, errors.New("resolved peer is not a channel")
	}

	return channel, nil
}

// getChannelMessages gets messages from a channel for the last 24 hours
func getChannelMessages(ctx context.Context, api *tg.Client, channel *tg.Channel) ([]*tg.Message, error) {
	oneDayAgo := time.Now().Add(-24 * time.Hour)
	messages, err := api.MessagesGetHistory(ctx, &tg.MessagesGetHistoryRequest{
		Peer: &tg.InputPeerChannel{
			ChannelID:  channel.ID,
			AccessHash: channel.AccessHash,
		},
		OffsetID:   0,
		OffsetDate: int(oneDayAgo.Unix()),
		AddOffset:  0,
		Limit:      100,
		MaxID:      0,
		MinID:      0,
		Hash:       0,
	})
	if err != nil {
		return nil, errors.Wrap(err, "get messages")
	}

	switch m := messages.(type) {
	case *tg.MessagesChannelMessages:
		var result []*tg.Message
		for _, msg := range m.Messages {
			if message, ok := msg.(*tg.Message); ok {
				result = append(result, message)
			}
		}
		return result, nil
	default:
		return nil, errors.Errorf("unexpected messages response type: %T", messages)
	}
}

func run(ctx context.Context) error {
	// Parse command line flags
	channelName, err := parseFlags()
	if err != nil {
		return err
	}

	// Load environment variables
	phone, appID, appHash, err := loadEnv()
	if err != nil {
		return err
	}

	// Initialize Telegram client
	client, waiter, err := initClient(phone, appID, appHash)
	if err != nil {
		return err
	}

	api := client.API()
	flow := auth.NewFlow(examples.Terminal{PhoneNumber: phone}, auth.SendCodeOptions{})

	// Run client and get dialogs
	return waiter.Run(ctx, func(ctx context.Context) error {
		if err := client.Run(ctx, func(ctx context.Context) error {
			// Perform auth if no session is available
			if err := client.Auth().IfNecessary(ctx, flow); err != nil {
				return errors.Wrap(err, "auth")
			}

			// Get channel
			channel, err := getChannel(ctx, api, channelName)
			if err != nil {
				return err
			}

			fmt.Printf("Found channel: %s\n", channel.Title)

			// Get messages
			messages, err := getChannelMessages(ctx, api, channel)
			if err != nil {
				return err
			}

			// Print messages
			fmt.Printf("\nMessages from the last 24 hours:\n")
			fmt.Println("============================")
			for _, message := range messages {
				msgTime := time.Unix(int64(message.Date), 0)
				fmt.Printf("[%s] %s\n", msgTime.Format("15:04:05"), message.Message)
			}

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
