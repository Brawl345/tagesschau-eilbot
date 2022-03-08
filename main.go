package main

import (
	"github.com/Brawl345/tagesschau-eilbot/handler"
	"github.com/Brawl345/tagesschau-eilbot/storage"
	"log"
	"time"

	"gopkg.in/telebot.v3"
)

func main() {
	var config = &storage.Config{}
	err := config.Load()

	if err != nil {
		log.Fatalln(err)
	}

	pref := telebot.Settings{
		Token:  config.Token,
		Poller: &telebot.LongPoller{Timeout: 10 * time.Second},
	}

	bot, err := telebot.NewBot(pref)
	if err != nil {
		log.Fatalln(err)
		return
	}

	log.Printf("Logged in as @%s (%d)", bot.Me.Username, bot.Me.ID)

	h := handler.Handler{
		Bot:    bot,
		Config: config,
	}

	time.AfterFunc(10*time.Second, h.OnTimer)

	bot.Handle("/help", h.OnHelp)
	bot.Handle("/hilfe", h.OnHelp)
	bot.Handle("/start", h.OnStart)
	bot.Handle("/stop", h.OnStop)

	bot.Start()
}
