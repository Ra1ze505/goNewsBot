package handlers

import (
	"fmt"
	"time"

	"github.com/Ra1ze505/goNewsBot/src/repository"
	tele "gopkg.in/telebot.v4"
)

func HelloHandle(userRepo repository.UserRepositoryInterface) tele.HandlerFunc {
	return func(context tele.Context) error {
		return helloHandle(context, userRepo)
	}
}

func helloHandle(context tele.Context, userRepo repository.UserRepositoryInterface) error {
	user := &repository.User{
		Username:    &context.Sender().Username,
		ChatID:      int64(context.Sender().ID),
		City:        "Москва",
		Timezone:    "+3",
		MailingTime: time.Date(0, 0, 0, 10, 0, 0, 0, time.Local),
	}

	err := userRepo.CreateOrUpdateUser(user)
	if err != nil {
		context.Send("Что-то пошло не так :(\nПопробуй позже")
		return err
	}

	context.Send(fmt.Sprintf("Привет, @%s", context.Sender().Username))

	return nil
}

