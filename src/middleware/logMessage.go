package middleware

import (
	log "github.com/sirupsen/logrus"
	tele "gopkg.in/telebot.v4"
)

func MessageLogger() tele.MiddlewareFunc {
	return func(next tele.HandlerFunc) tele.HandlerFunc {
		return func(c tele.Context) error {
			var (
				user = c.Sender()
				text = c.Text()
			)
			log.Infof("Got message: `%s` from user: @%s", text, user.Username)
			return next(c)
		}
	}
}
