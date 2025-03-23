package repository

import (
	"database/sql"
	"time"

	"github.com/pkg/errors"
)

type User struct {
	ID          int       `db:"id"`
	Username    *string   `db:"username"`
	ChatID      int64     `db:"chat_id"`
	CreatedAt   time.Time `db:"created_at"`
	City        string    `db:"city"`
	Timezone    string    `db:"timezone"`
	MailingTime time.Time `db:"mailing_time"`
}

type UserRepositoryInterface interface {
	CreateOrUpdateUser(user *User) error
	UpdateUser(user *User) error
	GetUsersByMailingTime(mailingTime time.Time) ([]*User, error)
}

type UserRepository struct {
	db *sql.DB
}

func NewUserRepository(db *sql.DB) UserRepositoryInterface {
	return &UserRepository{db: db}
}

func (r *UserRepository) CreateOrUpdateUser(user *User) error {
	stmt := `
	INSERT INTO users (username, chat_id, city, timezone, mailing_time)
	VALUES ($1, $2, $3, $4, $5) ON CONFLICT (chat_id) DO
	UPDATE
	SET username = EXCLUDED.username
	`
	_, err := r.db.Exec(stmt, user.Username, user.ChatID, user.City, user.Timezone, user.MailingTime)

	if err != nil {
		return errors.Wrap(err, "failed to create user")
	}
	return nil
}

func (r *UserRepository) UpdateUser(user *User) error {
	stmt := `UPDATE users SET username = ?, city = ?, timezone = ?, mailing_time = ? WHERE id = ?`
	_, err := r.db.Exec(stmt, user.Username, user.City, user.Timezone, user.MailingTime, user.ID)
	if err != nil {
		return errors.Wrap(err, "failed to update user")
	}
	return nil
}

func (r *UserRepository) GetUsersByMailingTime(mailingTime time.Time) ([]*User, error) {
	rows, err := r.db.Query("SELECT * FROM users WHERE mailing_time = ?", mailingTime)
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
