package middleware

import (
	"time"

	"github.com/Ra1ze505/goNewsBot/src/repository"
	tele "gopkg.in/telebot.v4"
)

func CreateOrUpdateUser(userRepo repository.UserRepositoryInterface) tele.MiddlewareFunc {
	return func(next tele.HandlerFunc) tele.HandlerFunc {
		return func(c tele.Context) error {
			return createOrUpdateUser(c, next, userRepo)
		}
	}
}

func createOrUpdateUser(c tele.Context, next tele.HandlerFunc, userRepo repository.UserRepositoryInterface) error {
	user := &repository.User{
		Username:           &c.Sender().Username,
		ChatID:             c.Sender().ID,
		City:               "Москва",
		MailingTime:        time.Date(0, 0, 0, 7, 0, 0, 0, time.UTC),
		PreferredChannelID: 1429590454,
	}

	updatedUser, err := userRepo.CreateOrUpdateUser(user)
	if err != nil {
		c.Send("Что-то пошло не так :(\nПопробуй позже")
		return err
	}

	c.Set("user", updatedUser)

	if c.Callback() != nil && c.Callback().Data == "change_city" {
		c.Set("changing_city", true)
	}

	return next(c)
}
