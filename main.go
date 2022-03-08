package main

import (
	"github.com/Brawl345/tagesschau-eilbot/handler"
	"github.com/Brawl345/tagesschau-eilbot/storage"
	_ "github.com/joho/godotenv/autoload"
	"log"
	"os"
	"time"

	"gopkg.in/telebot.v3"
)

func main() {
	db, err := storage.Open(os.Getenv("TAGESSCHAU_EILBOT_MYSQL_URL"))
	if err != nil {
		log.Fatal(err)
	}

	if err := db.Ping(); err != nil {
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
		Token:  os.Getenv("TAGESSCHAU_EILBOT_TOKEN"),
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

	bot.Start()
}
