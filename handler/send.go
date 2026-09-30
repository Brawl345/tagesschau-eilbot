package handler

import (
	"context"
	"errors"
	"fmt"
	"log"
	"sync"
	"time"

	"gopkg.in/telebot.v3"
)

const (
	sendWorkers  = 8
	sendInterval = time.Second / 25
	maxAttempts  = 3
)

type messageFunc func(chatId int64) (string, *telebot.SendOptions)

// sendAll sends to all chats in parallel while staying below Telegram's
// broadcast limit of about 30 messages per second.
func (h *Handler) sendAll(ctx context.Context, chatIds []int64, message messageFunc) {
	ticker := time.NewTicker(sendInterval)
	defer ticker.Stop()

	jobs := make(chan int64)
	var wg sync.WaitGroup
	for range min(sendWorkers, len(chatIds)) {
		wg.Go(func() {
			for chatId := range jobs {
				if err := h.send(ctx, ticker.C, chatId, message); err != nil {
					log.Printf("Error for subscriber %d: %s", chatId, err)
				}
			}
		})
	}

	for _, chatId := range chatIds {
		jobs <- chatId
	}
	close(jobs)
	wg.Wait()
}

func (h *Handler) send(ctx context.Context, tick <-chan time.Time, chatId int64, message messageFunc) error {
	text, options := message(chatId)

	for range maxAttempts {
		<-tick
		_, err := h.Bot.Send(telebot.ChatID(chatId), text, options)
		if err == nil {
			return nil
		}

		var floodErr telebot.FloodError
		var groupErr telebot.GroupError
		var tgErr *telebot.Error

		switch {
		case errors.As(err, &floodErr):
			log.Printf("%d: Flood error, retrying after %d seconds", chatId, floodErr.RetryAfter)
			time.Sleep(time.Duration(floodErr.RetryAfter+1) * time.Second)

		case errors.As(err, &groupErr):
			log.Printf("Chat %d migrated to new group %d", chatId, groupErr.MigratedTo)
			if _, err := h.DB.Subscribers.Delete(ctx, chatId); err != nil {
				return fmt.Errorf("deleting migrated chat: %w", err)
			}
			isNew, err := h.DB.Subscribers.Create(ctx, groupErr.MigratedTo)
			if err != nil {
				return fmt.Errorf("creating migrated chat: %w", err)
			}
			if !isNew {
				return nil
			}
			chatId = groupErr.MigratedTo

		case errors.Is(err, telebot.ErrChatNotFound), errors.As(err, &tgErr) && tgErr.Code == 403:
			log.Printf("%d: %s, will be removed", chatId, err)
			if _, err := h.DB.Subscribers.Delete(ctx, chatId); err != nil {
				return fmt.Errorf("deleting unreachable chat: %w", err)
			}
			return nil

		default:
			return err
		}
	}

	return fmt.Errorf("giving up after %d attempts", maxAttempts)
}
