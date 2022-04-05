package main

import (
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/Brawl345/tagesschau-eilbot/handler"
	"github.com/Brawl345/tagesschau-eilbot/storage"
	_ "github.com/joho/godotenv/autoload"

	"gopkg.in/telebot.v3"
)

func main() {
	db, err := storage.Connect()
	if err != nil {
		log.Fatal(err)
	}

	log.Println("Database connection established")

	n, err := db.Migrate()
	if err != nil {
		log.Fatalln(err)
	}
	if n > 0 {
		log.Printf("Applied %d migration(s)", n)
	}

	pref := telebot.Settings{
		Token:  os.Getenv("BOT_TOKEN"),
		Poller: &telebot.LongPoller{Timeout: 10 * time.Second},
	}

	bot, err := telebot.NewBot(pref)
	if err != nil {
		log.Fatalln(err)
	}

	log.Printf("Logged in as @%s (%d)", bot.Me.Username, bot.Me.ID)

	h := handler.Handler{
		Bot: bot,
		DB:  db,
	}

	time.AfterFunc(5*time.Second, h.OnTimer)

	bot.Handle("/help", h.OnHelp)
	bot.Handle("/hilfe", h.OnHelp)
	bot.Handle("/start", h.OnStart)
	bot.Handle("/stop", h.OnStop)

	channel := make(chan os.Signal)
	signal.Notify(channel, os.Interrupt, syscall.SIGTERM)
	signal.Notify(channel, os.Interrupt, syscall.SIGKILL)
	signal.Notify(channel, os.Interrupt, syscall.SIGINT)
	go func() {
		<-channel
		log.Println("Stopping...")
		bot.Stop()
		err := db.Close()
		if err != nil {
			log.Println(err)
			os.Exit(1)
			return
		}
		os.Exit(0)
	}()

	bot.Start()
}
