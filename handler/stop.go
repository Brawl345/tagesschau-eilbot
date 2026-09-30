package handler

import (
	"context"
	"log"
	"strings"

	"gopkg.in/telebot.v3"
)

func (h *Handler) OnStop(c telebot.Context) error {
	if !isAuthorized(c) {
		return c.Send("❌ Nur Gruppenadministratoren können Eilmeldungen deabonnieren.", defaultSendOptions)
	}

	chatId := c.Chat().ID
	sb := strings.Builder{}

	deleted, err := h.DB.Subscribers.Delete(context.Background(), chatId)
	if err != nil {
		log.Println(err)
		return c.Send("❌ Beim Deabonnieren ist ein Fehler aufgetreten.", defaultSendOptions)
	}

	if !deleted {
		sb.WriteString("<b>❌ Eilmeldungen wurden noch nicht abonniert.</b>\n")
		sb.WriteString("Nutze /start zum Abonnieren.")
		return c.Send(sb.String(), defaultSendOptions)
	}

	log.Println("Removed subscription:", chatId)

	sb.WriteString("<b>✅ Du erhältst jetzt keine Eilmeldungen mehr.</b>\n")
	sb.WriteString("Nutze /start, um wieder Eilmeldungen zu erhalten.\n")

	return c.Send(sb.String(), defaultSendOptions)
}
