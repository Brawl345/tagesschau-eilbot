package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"github.com/Brawl345/tagesschau-eilbot/handler"
	"github.com/Brawl345/tagesschau-eilbot/storage"
	_ "github.com/joho/godotenv/autoload"

	"gopkg.in/telebot.v3"
)

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	db, err := storage.Connect()
	if err != nil {
		log.Fatalln(err)
	}
	defer func() {
		if err := db.Close(); err != nil {
			log.Println(err)
		}
	}()

	log.Println("Database connection established")

	n, err := db.Migrate()
	if err != nil {
		log.Fatalln(err)
	}
	if n > 0 {
		log.Printf("Applied %d migration(s)", n)
	}

	bot, err := telebot.NewBot(telebot.Settings{
		Token:  os.Getenv("BOT_TOKEN"),
		Poller: &telebot.LongPoller{Timeout: 10 * time.Second},
	})
	if err != nil {
		log.Fatalln(err)
	}

	log.Printf("Logged in as @%s (%d)", bot.Me.Username, bot.Me.ID)

	h := handler.New(bot, db)

	bot.Handle("/help", h.OnHelp)
	bot.Handle("/hilfe", h.OnHelp)
	bot.Handle("/start", h.OnStart)
	bot.Handle("/stop", h.OnStop)

	var wg sync.WaitGroup
	wg.Go(func() { h.Poll(ctx) })
	wg.Go(func() {
		<-ctx.Done()
		log.Println("Stopping...")
		bot.Stop()
	})

	bot.Start()
	wg.Wait()
}
