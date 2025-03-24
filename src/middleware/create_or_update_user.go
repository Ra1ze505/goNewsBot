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
		Username:    &c.Sender().Username,
		ChatID:      c.Sender().ID,
		City:        "Москва",
		Timezone:    "+3",
		MailingTime: time.Date(0, 0, 0, 10, 0, 0, 0, time.Local),
	}

	err := userRepo.CreateOrUpdateUser(user)
	if err != nil {
		c.Send("Что-то пошло не так :(\nПопробуй позже")
		return err
	}

	return next(c)
}
