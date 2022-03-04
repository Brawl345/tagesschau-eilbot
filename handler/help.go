package handler

import "gopkg.in/telebot.v3"

func (h Handler) OnHelp(c telebot.Context) error {
	text := "<b>Inoffizieller Tagesschau-Eilmeldungen-Bot</b>\n"
	text += "<b>/start:</b> Eilmeldungen erhalten\n"
	text += "<b>/stop:</b> Eilmeldungen nicht mehr erhalten"

	return c.Send(text, defaultSendOptions)
}
