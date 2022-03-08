package handler

import (
	"github.com/Brawl345/tagesschau-eilbot/storage"
	"gopkg.in/telebot.v3"
)

type Handler struct {
	Bot *telebot.Bot
	DB  *storage.DB
}
