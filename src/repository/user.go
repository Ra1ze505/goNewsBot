package repository

//go:generate mockgen -source=user.go -destination=../mocks/repository/user_mock.go

import (
	"database/sql"
	"time"

	"github.com/pkg/errors"
)

type User struct {
	ID                 *int      `db:"id"`
	Username           *string   `db:"username"`
	ChatID             int64     `db:"chat_id"`
	CreatedAt          time.Time `db:"created_at"`
	City               string    `db:"city"`
	Timezone           string    `db:"timezone"`
	MailingTime        time.Time `db:"mailing_time"`
	PreferredChannelID int64     `db:"preferred_channel_id"`
}

type UserRepositoryInterface interface {
	CreateOrUpdateUser(user *User) (*User, error)
	GetUsersByMailingTime(mailingTime time.Time) ([]*User, error)
	UpdateUserCityAndTimezone(userID *int, city string, timezone string) error
	UpdatePreferredChannel(userID *int, channelID int64) error
}

type UserRepository struct {
	db *sql.DB
}

func NewUserRepository(db *sql.DB) UserRepositoryInterface {
	return &UserRepository{db: db}
}

func (r *UserRepository) CreateOrUpdateUser(user *User) (*User, error) {
	var existingUser User

	err := r.db.QueryRow(`
		SELECT id, username, chat_id, city, timezone, mailing_time, preferred_channel_id
		FROM users
		WHERE chat_id = $1
	`, user.ChatID).Scan(&existingUser.ID, &existingUser.Username, &existingUser.ChatID, &existingUser.City, &existingUser.Timezone, &existingUser.MailingTime, &existingUser.PreferredChannelID)
	if err == sql.ErrNoRows {
		var id int
		err = r.db.QueryRow(`
			INSERT INTO users (username, chat_id, city, timezone, mailing_time, preferred_channel_id)
			VALUES ($1, $2, $3, $4, $5, $6)
			RETURNING id
		`, user.Username, user.ChatID, user.City, user.Timezone, user.MailingTime, user.PreferredChannelID).Scan(&id)

		if err != nil {
			return nil, errors.Wrap(err, "failed to create user")
		}

		return &User{
			ID:                 &id,
			Username:           user.Username,
			ChatID:             user.ChatID,
			City:               user.City,
			Timezone:           user.Timezone,
			MailingTime:        user.MailingTime,
			PreferredChannelID: user.PreferredChannelID,
		}, nil
	}

	if err != nil {
		return nil, errors.Wrap(err, "failed to check existing user")
	}

	if existingUser.Username == nil || *existingUser.Username != *user.Username {
		_, err = r.db.Exec(`
			UPDATE users
			SET username = $1
			WHERE chat_id = $2
		`, user.Username, user.ChatID)

		if err != nil {
			return nil, errors.Wrap(err, "failed to update user")
		}
	}

	return &existingUser, nil
}

func (r *UserRepository) GetUsersByMailingTime(mailingTime time.Time) ([]*User, error) {
	rows, err := r.db.Query("SELECT * FROM users WHERE mailing_time = $1", mailingTime)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get users by mailing time")
	}
	defer rows.Close()

	var users []*User
	for rows.Next() {
		user := &User{}
		err := rows.Scan(&user.ID, user.Username, &user.ChatID, &user.CreatedAt, &user.City, &user.Timezone, &user.MailingTime)
		if err != nil {
			return nil, errors.Wrap(err, "failed to scan user")
		}
		users = append(users, user)
	}

	return users, nil
}

func (r *UserRepository) UpdateUserCityAndTimezone(userID *int, city string, timezone string) error {
	if userID == nil {
		return errors.New("user ID is nil")
	}
	stmt := `UPDATE users SET city = $1, timezone = $2 WHERE id = $3`
	_, err := r.db.Exec(stmt, city, timezone, *userID)
	if err != nil {
		return errors.Wrap(err, "failed to update user city and timezone")
	}
	return nil
}

func (r *UserRepository) UpdatePreferredChannel(userID *int, channelID int64) error {
	if userID == nil {
		return errors.New("user ID is nil")
	}
	stmt := `UPDATE users SET preferred_channel_id = $1 WHERE id = $2`
	_, err := r.db.Exec(stmt, channelID, *userID)
	if err != nil {
		return errors.Wrap(err, "failed to update user preferred channel")
	}
	return nil
}
