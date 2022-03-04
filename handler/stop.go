package handler

import (
	"gopkg.in/telebot.v3"
	"log"
)

func (h Handler) OnStop(c telebot.Context) error {
	if c.Chat().Type != telebot.ChatPrivate {
		if !isGroupAdmin(h.Bot, c.Chat().ID, c.Message().Sender.ID) {
			return c.Send("❌ Nur Gruppenadministratoren können Eilmeldungen deabonnieren.", defaultSendOptions)
		}
	}

	chatId := c.Chat().ID

	var text string

	err := h.Config.RemoveSubscriber(chatId)

	if err == nil {
		log.Println("Removed subscription:", chatId)
		err := h.Config.Save()

		if err != nil {
			log.Println("Failed writing config:", err)
			return c.Send("❌ Beim Deabonnieren ist ein Fehler aufgetreten.", defaultSendOptions)
		}

		text = "<b>✅ Du erhältst jetzt keine Eilmeldungen mehr.</b>\n"
		text += "Nutze /start, um wieder Eilmeldungen zu erhalten.\n"
	} else {
		text = "<b>❌ Eilmeldungen wurden noch nicht abonniert.</b>\n"
		text += "Nutze /start zum Abonnieren."
	}

	return c.Send(text, defaultSendOptions)

}
