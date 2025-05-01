package repository

import (
	"database/sql"
	"time"
)

type Message struct {
	ID          int64
	ChannelID   int64
	MessageID   int
	MessageText string
	MessageDate time.Time
	CreatedAt   time.Time
}

type MessageRepositoryInterface interface {
	SaveMessage(message *Message) error
	GetLastMessageTime(channelID int64) (time.Time, error)
}

type MessageRepository struct {
	db *sql.DB
}

func NewMessageRepository(db *sql.DB) MessageRepositoryInterface {
	return &MessageRepository{db: db}
}

func (r *MessageRepository) SaveMessage(message *Message) error {
	query := `
		INSERT INTO messages (channel_id, message_id, message_text, message_date)
		VALUES ($1, $2, $3, $4)
		ON CONFLICT (channel_id, message_id) DO NOTHING
	`
	_, err := r.db.Exec(query,
		message.ChannelID,
		message.MessageID,
		message.MessageText,
		message.MessageDate,
	)
	return err
}

func (r *MessageRepository) GetLastMessageTime(channelID int64) (time.Time, error) {
	var messageDate time.Time
	query := `
		SELECT message_date 
		FROM messages 
		WHERE channel_id = $1 
		ORDER BY message_date DESC 
		LIMIT 1
	`
	err := r.db.QueryRow(query, channelID).Scan(&messageDate)
	if err == sql.ErrNoRows {
		return time.Time{}, nil
	}
	return messageDate.UTC(), err
}
