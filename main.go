package main

import (
	"github.com/Brawl345/tagesschau-eilbot/handler"
	"github.com/Brawl345/tagesschau-eilbot/storage"
	"github.com/robfig/cron/v3"
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

	c := cron.New()
	c.AddFunc("@every 1m", h.OnCheck)
	c.Start()

	bot.Handle("/help", h.OnHelp)
	bot.Handle("/hilfe", h.OnHelp)
	bot.Handle("/start", h.OnStart)
	bot.Handle("/stop", h.OnStop)

	bot.Start()
}
