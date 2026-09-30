package handler

import (
	"net/http"
	"time"

	"github.com/Brawl345/tagesschau-eilbot/storage"
	"gopkg.in/telebot.v3"
)

type Handler struct {
	Bot        *telebot.Bot
	DB         *storage.DB
	httpClient *http.Client
}

func New(bot *telebot.Bot, db *storage.DB) *Handler {
	return &Handler{
		Bot:        bot,
		DB:         db,
		httpClient: &http.Client{Timeout: 30 * time.Second},
	}
}
