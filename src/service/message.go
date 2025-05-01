package service

import (
	"context"
	"database/sql"
	"os"
	"path/filepath"
	"strconv"
	"time"

	"github.com/Ra1ze505/goNewsBot/src/config"
	"github.com/Ra1ze505/goNewsBot/src/repository"

	"github.com/gotd/contrib/middleware/floodwait"
	"github.com/gotd/td/telegram"
	"github.com/gotd/td/tg"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
	"go.uber.org/zap"
)

const (
	MessageBatchLimit = 20
)

type MessageService struct {
	client   *telegram.Client
	api      *tg.Client
	repo     repository.MessageRepositoryInterface
	channels []string
	// Channel to signal when messages are fetched
	MessagesFetched chan struct{}
}

func NewMessageService(client *telegram.Client, repo repository.MessageRepositoryInterface) *MessageService {
	return &MessageService{
		client:          client,
		api:             client.API(),
		repo:            repo,
		channels:        config.Channels,
		MessagesFetched: make(chan struct{}),
	}
}

func (s *MessageService) StartMessageFetcher(ctx context.Context) {
	ticker := time.NewTicker(1 * time.Hour)
	defer ticker.Stop()

	if err := s.fetchMessages(ctx); err != nil {
		log.Errorf("Error fetching messages on startup: %v", err)
	} else {
		// Signal that messages were fetched successfully
		s.MessagesFetched <- struct{}{}
	}

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			if err := s.fetchMessages(ctx); err != nil {
				log.Errorf("Error fetching messages: %v", err)
			} else {
				// Signal that messages were fetched successfully
				s.MessagesFetched <- struct{}{}
			}
		}
	}
}

func (s *MessageService) fetchMessages(ctx context.Context) error {
	log.Info("Fetching messages")
	for _, channelUsername := range s.channels {
		log.Infof("Fetching messages for channel: %s", channelUsername)
		channel, err := s.getChannel(ctx, channelUsername)
		if err != nil {
			log.Errorf("Error resolving channel: %v", err)
			continue
		}

		lastMessageTime, err := s.repo.GetLastMessageTime(channel.ID)
		if err != nil {
			log.Errorf("Error getting last message time: %v", err)
			continue
		}

		messages, err := s.getChannelMessages(ctx, channel, lastMessageTime)
		if err != nil {
			log.Errorf("Error getting channel messages: %v", err)
			continue
		}

		log.Infof("Fetched %d messages for channel: %s", len(messages), channelUsername)

		for _, msg := range messages {
			message := &repository.Message{
				ChannelID:       channel.ID,
				MessageID:       msg.ID,
				ChannelUsername: channelUsername,
				MessageText:     msg.Message,
				MessageDate:     time.Unix(int64(msg.Date), 0),
			}

			if err := s.repo.SaveMessage(message); err != nil {
				log.Errorf("Error saving message: %v", err)
				continue
			}
		}
	}
	return nil
}

func (s *MessageService) getChannel(ctx context.Context, username string) (*tg.Channel, error) {
	resolved, err := s.api.ContactsResolveUsername(ctx, &tg.ContactsResolveUsernameRequest{
		Username: username,
	})

	if err != nil {
		log.Errorf("Error resolving username: %v", err)
		return nil, err
	}

	channel, ok := resolved.Chats[0].(*tg.Channel)
	if !ok {
		log.Error("Channel not found or not a channel")
		return nil, errors.New("channel not found")
	}

	return channel, nil
}

func (s *MessageService) getChannelMessages(ctx context.Context, channel *tg.Channel, lastMessageTime time.Time) ([]*tg.Message, error) {
	oneDayAgo := time.Now().Add(-24 * time.Hour)
	oneDayAgoTimestamp := int(oneDayAgo.Unix())

	var allMessages []*tg.Message
	processedMessageIDs := make(map[int]bool)

	offsetID := 0

	if lastMessageTime.IsZero() {
		lastMessageTime = oneDayAgo
	}

	for {
		messages, err := s.api.MessagesGetHistory(ctx, &tg.MessagesGetHistoryRequest{
			Peer: &tg.InputPeerChannel{
				ChannelID:  channel.ID,
				AccessHash: channel.AccessHash,
			},
			OffsetID: offsetID,
			Limit:    MessageBatchLimit,
			MaxID:    0,
			MinID:    0,
			Hash:     0,
		})
		if err != nil {
			log.Errorf("Error getting messages: %v", err)
			return nil, err
		}

		channelMessages, ok := messages.(*tg.MessagesChannelMessages)
		if !ok {
			log.Errorf("Unexpected messages response type: %T", messages)
			return nil, errors.New("unexpected messages response type")
		}

		if len(channelMessages.Messages) == 0 {
			return allMessages, nil
		}

		var batchMessages []*tg.Message
		var shouldStop bool

		for _, msg := range channelMessages.Messages {
			if message, ok := msg.(*tg.Message); ok {
				offsetID = message.ID

				messageTime := time.Unix(int64(message.Date), 0)

				if message.Date < oneDayAgoTimestamp {
					shouldStop = true
					break
				}

				if messageTime.Before(lastMessageTime) {
					shouldStop = true
					break
				}

				if processedMessageIDs[message.ID] {
					continue
				}

				if message.Message == "" {
					continue
				}

				batchMessages = append(batchMessages, message)
				processedMessageIDs[message.ID] = true
			}
		}

		allMessages = append(allMessages, batchMessages...)

		if shouldStop {
			return allMessages, nil
		}

		if len(channelMessages.Messages) < MessageBatchLimit {
			log.Infof("Received less than %d messages, reached the end", MessageBatchLimit)
			return allMessages, nil
		}
	}
}

func InitAndStartMessageService(ctx context.Context, db *sql.DB) (*MessageService, error) {
	appID, err := strconv.Atoi(os.Getenv("API_ID"))
	if err != nil {
		return nil, errors.Wrap(err, "parse app id")
	}
	appHash := os.Getenv("API_HASH")
	if appHash == "" {
		return nil, errors.New("no app hash")
	}

	phone := os.Getenv("TG_PHONE")
	if phone == "" {
		return nil, errors.New("no phone number")
	}

	sessionDir := config.SessionDir
	waiter := floodwait.NewWaiter().WithCallback(func(ctx context.Context, wait floodwait.FloodWait) {
		log.Infof("Got FLOOD_WAIT. Will retry after %v", wait.Duration)
	})

	client := telegram.NewClient(appID, appHash, telegram.Options{
		Logger: zap.NewNop(),
		SessionStorage: &telegram.FileSessionStorage{
			Path: filepath.Join(sessionDir, "session.json"),
		},
		Middlewares: []telegram.Middleware{
			waiter,
		},
	})

	messageRepo := repository.NewMessageRepository(db)
	messageService := NewMessageService(client, messageRepo)

	go func() {
		if err := waiter.Run(ctx, func(ctx context.Context) error {
			if err := client.Run(ctx, func(ctx context.Context) error {
				messageService.StartMessageFetcher(ctx)
				return nil
			}); err != nil {
				return errors.Wrap(err, "run")
			}
			return nil
		}); err != nil {
			log.Errorf("Error running Telegram client: %v", err)
		}
	}()

	return messageService, nil
}
