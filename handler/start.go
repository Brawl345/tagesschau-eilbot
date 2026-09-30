package handler

import (
	"context"
	"log"
	"strings"

	"gopkg.in/telebot.v3"
)

func (h *Handler) OnStart(c telebot.Context) error {
	if !isAuthorized(c) {
		return c.Send("❌ Nur Gruppenadministratoren können Eilmeldungen abonnieren.", defaultSendOptions)
	}

	chatId := c.Chat().ID
	sb := strings.Builder{}

	isNew, err := h.DB.Subscribers.Create(context.Background(), chatId)
	if err != nil {
		log.Println(err)
		return c.Send("❌ Beim Abonnieren ist ein Fehler aufgetreten.", defaultSendOptions)
	}

	if !isNew {
		sb.WriteString("<b>✅ Du erhältst bereits Eilmeldungen.</b>\n")
		sb.WriteString("Nutze /stop zum Deabonnieren.")
		return c.Send(sb.String(), defaultSendOptions)
	}

	log.Println("New subscriber:", chatId)

	sb.WriteString("<b>✅ Du erhältst jetzt neue Eilmeldungen!</b>\n")
	sb.WriteString("Nutze /stop, um keine Eilmeldungen mehr zu erhalten.\n")
	sb.WriteString("Für neue Tagesschau-Artikel, abonniere den @TagesschauDE-Kanal.\n\n")

	sb.WriteString("<b>ACHTUNG:</b> ")
	if c.Message().Private() {
		sb.WriteString("Wenn du den Bot blockierst, musst du die Eilmeldungen erneut abonnieren!")
	} else {
		sb.WriteString("Wenn du den Bot aus der Gruppe entfernst, musst du die Eilmeldungen erneut abonnieren!")
	}

	return c.Send(sb.String(), defaultSendOptions)
}
