package adminhandlers

import (
	"fmt"
	"os"
	"strconv"

	tele "gopkg.in/telebot.v4"

	"github.com/Ra1ze505/goNewsBot/src/config"
	"github.com/Ra1ze505/goNewsBot/src/keyboard"
	"github.com/Ra1ze505/goNewsBot/src/repository"
	log "github.com/sirupsen/logrus"
)

type AdminHandler struct {
	ForceRegenerateChannel chan struct{}
	userRepo               repository.UserRepositoryInterface
	summaryRepo            repository.SummaryRepositoryInterface
}

func NewAdminHandler(userRepo repository.UserRepositoryInterface, summaryRepo repository.SummaryRepositoryInterface) *AdminHandler {
	return &AdminHandler{
		ForceRegenerateChannel: make(chan struct{}),
		userRepo:               userRepo,
		summaryRepo:            summaryRepo,
	}
}

func (h *AdminHandler) Handle(c tele.Context) error {
	if !isAdmin(c) {
		return c.Send("Вы не являетесь администратором", keyboard.GetStartKeyboard())
	}

	var rows []tele.Row
	rows = append(rows, tele.Row{tele.Btn{
		Text: "Перенерировать суммаризацию",
		Data: "admin_regenerate_summary",
	}})

	k := &tele.ReplyMarkup{
		ResizeKeyboard: true,
	}
	k.Inline(rows...)

	return c.Send("Выберите действие", k)
}

func (h *AdminHandler) HandleRegenerateSummary(c tele.Context) error {
	if !isAdmin(c) {
		return c.Send("Вы не являетесь администратором", keyboard.GetStartKeyboard())
	}

	var rows []tele.Row
	for channelID, channelName := range config.Channels {
		btn := tele.Btn{
			Text: channelName,
			Data: fmt.Sprintf("regenerate_summary_%d", channelID),
		}
		rows = append(rows, tele.Row{btn})
	}

	k := &tele.ReplyMarkup{
		ResizeKeyboard: true,
	}
	k.Inline(rows...)

	return c.Send("Выберите канал", k)
}

func (h *AdminHandler) HandleRegenerateSummaryChannel(c tele.Context) error {
	if !isAdmin(c) {
		return c.Send("Вы не являетесь администратором", keyboard.GetStartKeyboard())
	}

	var channelID int64
	_, err := fmt.Sscanf(c.Callback().Data, "regenerate_summary_%d", &channelID)
	if err != nil {
		log.Infof("error parsing channel ID: %v", err)
		return c.Send("Некорректный канал", keyboard.GetStartKeyboard())
	}

	err = h.summaryRepo.DeleteLastSummary(channelID)
	if err != nil {
		log.Infof("error deleting summary: %v", err)
		return c.Send("Не удалось удалить суммаризацию", keyboard.GetStartKeyboard())
	}

	go func() {
		h.ForceRegenerateChannel <- struct{}{}
		log.Infof("force regenerate channel signal sent")
	}()

	return c.Send("Суммаризация удалена, суммаризатор будет запущен в ближайшее время", keyboard.GetStartKeyboard())
}

func isAdmin(c tele.Context) bool {
	user, ok := c.Get("user").(*repository.User)
	if !ok {
		return false
	}

	adminChatID, err := strconv.ParseInt(os.Getenv("ADMIN_ID"), 10, 64)
	if err != nil {
		log.Infof("error parsing ADMIN_ID: %v", err)
		return false
	}

	return user.ChatID == adminChatID
}
