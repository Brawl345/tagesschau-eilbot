package handler

import (
	"gopkg.in/telebot.v3"
	"log"
)

func (h Handler) OnStart(c telebot.Context) error {
	if c.Chat().Type != telebot.ChatPrivate {
		if !isGroupAdmin(h.Bot, c.Chat().ID, c.Message().Sender.ID) {
			return c.Send("❌ Nur Gruppenadministratoren können Eilmeldungen abonnieren.", defaultSendOptions)
		}
	}

	chatId := c.Chat().ID

	var text string
	err := h.Config.AddSubscriber(chatId)

	if err != nil {
		text = "<b>✅ Du erhältst bereits Eilmeldungen.</b>\n"
		text += "Nutze /stop zum Deabonnieren."
	} else {
		log.Println("New subscriber:", chatId)
		err := h.Config.Save()

		if err != nil {
			log.Println("Failed writing config:", err)
			return c.Send("❌ Beim Abonnieren ist ein Fehler aufgetreten.", defaultSendOptions)
		}

		text = "<b>✅ Du erhältst jetzt neue Eilmeldungen!</b>\n"
		text += "Nutze /stop, um keine Eilmeldungen mehr zu erhalten.\n"
		text += "Für neue Tagesschau-Artikel, abonniere den @TagesschauDE-Kanal.\n\n"

		text += "<b>ACHTUNG:</b> "
		if c.Chat().Type == telebot.ChatPrivate {
			text += "Wenn du den Bot blockierst, musst du die Eilmeldungen erneut abonnieren!"
		} else {
			text += "Wenn du den Bot aus der Gruppe entfernst, musst du die Eilmeldungen erneut abonnieren!"
		}
	}

	return c.Send(text, defaultSendOptions)
}
