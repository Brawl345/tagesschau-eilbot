package handler

import (
	"gopkg.in/telebot.v3"
	"log"
	"strings"
)

func (h Handler) OnStop(c telebot.Context) error {
	if c.Message().FromGroup() {
		if !isGroupAdmin(h.Bot, c.Chat().ID, c.Message().Sender.ID) {
			return c.Send("❌ Nur Gruppenadministratoren können Eilmeldungen deabonnieren.", defaultSendOptions)
		}
	}

	chatId := c.Chat().ID
	sb := strings.Builder{}

	exists, _ := h.DB.Subscribers.Exists(chatId)
	if !exists {
		sb.WriteString("<b>❌ Eilmeldungen wurden noch nicht abonniert.</b>\n")
		sb.WriteString("Nutze /start zum Abonnieren.")
		return c.Send(sb.String(), defaultSendOptions)
	}

	err := h.DB.Subscribers.Delete(chatId)
	if err != nil {
		log.Println(err)
		return c.Send("❌ Beim Deabonnieren ist ein Fehler aufgetreten.", defaultSendOptions)
	}

	log.Println("Removed subscription:", chatId)

	sb.WriteString("<b>✅ Du erhältst jetzt keine Eilmeldungen mehr.</b>\n")
	sb.WriteString("Nutze /start, um wieder Eilmeldungen zu erhalten.\n")

	return c.Send(sb.String(), defaultSendOptions)
}
