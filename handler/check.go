package handler

import (
	"encoding/json"
	"errors"
	"fmt"
	"html"
	"io"
	"log"
	"net/http"
	"strings"
	"time"

	"gopkg.in/telebot.v3"
)

const ApiUrl string = "https://www.tagesschau.de/json/headerapp"

type TagesschauResponse struct {
	BreakingNews []struct {
		Id       string `json:"id"`
		Headline string `json:"headline"`
		Text     string `json:"text"`
		Url      string `json:"url"`
		Date     string `json:"date"`
	} `json:"breakingNews"`
}

func (h Handler) OnTimer() {
	if isDebugMode() {
		log.Println("Checking for breaking news")
	}

	err := h.check()
	if err != nil {
		log.Println(err)
	}

	time.AfterFunc(1*time.Minute, h.OnTimer)
}

func (h Handler) check() error {
	resp, err := http.Get(ApiUrl)
	if err != nil {
		return err
	}

	if resp.StatusCode != 200 {
		return fmt.Errorf("got HTTP error %s", resp.Status)
	}

	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)

	if err != nil {
		return err
	}

	var result TagesschauResponse
	if err := json.Unmarshal(body, &result); err != nil {
		return fmt.Errorf("can not unmarshal JSON: %w", err)
	}

	if len(result.BreakingNews) == 0 || result.BreakingNews[0].Id == "" {
		if isDebugMode() {
			log.Println("No breaking news found")
		}
		return nil
	}

	breakingNews := result.BreakingNews[0]

	if breakingNews.Url == "" {
		return errors.New("invalid breaking news")
	}

	lastEntry, err := h.DB.System.GetLastEntry()
	if err != nil {
		return fmt.Errorf("error while getting last entry: %w", err)
	}

	if lastEntry == breakingNews.Id {
		if isDebugMode() {
			log.Println("Already notified of this breaking news")
		}
		return nil
	}

	log.Println("New breaking news found")

	sb := strings.Builder{}

	sb.WriteString(fmt.Sprintf("<b>%s</b>\n", html.EscapeString(strings.TrimSpace(breakingNews.Headline))))
	sb.WriteString(fmt.Sprintf("<i>%s</i>\n", html.EscapeString(strings.Replace(breakingNews.Date, "Stand: ", "", 1))))
	if breakingNews.Text != "" {
		sb.WriteString(fmt.Sprintf("%s\n", html.EscapeString(strings.TrimSpace(breakingNews.Text))))
	}

	url := breakingNews.Url
	if !strings.HasPrefix(url, "http") {
		url = "https://www.tagesschau.de/" + strings.TrimPrefix(url, "/")
	}

	textLink := fmt.Sprintf("<a href=\"%s\">Eilmeldung aufrufen</a>", url)
	replyMarkup := h.Bot.NewMarkup()
	btn := replyMarkup.URL("Eilmeldung aufrufen", url)
	replyMarkup.Inline(replyMarkup.Row(btn))

	groupText := "#EIL: " + sb.String()
	privateText := sb.String() + textLink

	subscribers, err := h.DB.Subscribers.GetAll()
	if err != nil {
		return fmt.Errorf("error while getting subscribers: %w", err)
	}

	for _, subscriber := range subscribers {
		if subscriber < 0 { // Group
			err = h.sendText(subscriber, groupText, &telebot.SendOptions{
				DisableWebPagePreview: true,
				ParseMode:             telebot.ModeHTML,
				ReplyMarkup:           replyMarkup,
			})
		} else {
			err = h.sendText(subscriber, privateText, defaultSendOptions)
		}

		if err != nil {
			log.Printf("Error for subscriber %d: %s", subscriber, err)
		}
	}

	err = h.DB.System.SetLastEntry(breakingNews.Id)
	if err != nil {
		return fmt.Errorf("failed writing last entry to DB: %w", err)
	}

	return nil
}

func (h Handler) sendText(subscriber int64, text string, sendOptions *telebot.SendOptions) error {
	_, err := h.Bot.Send(telebot.ChatID(subscriber), text, sendOptions)

	var telebotError *telebot.Error
	var floodError *telebot.FloodError

	if err != nil {
		if errors.Is(err, telebot.ErrChatNotFound) {
			log.Printf("Chat %d not found, will be deleted", subscriber)
			h.DB.Subscribers.Delete(subscriber)
		} else if errors.Is(err, telebot.ErrGroupMigrated) {
			migratedTo := err.(*telebot.GroupError).MigratedTo
			log.Printf("Chat %d migrated to new group %d", subscriber, migratedTo)
			h.DB.Subscribers.Delete(subscriber)
			h.DB.Subscribers.Create(migratedTo)
			return h.sendText(migratedTo, text, sendOptions)
		} else if errors.As(err, &floodError) {
			retryAfter := floodError.RetryAfter
			log.Printf("%d: Flood error, retrying after: %d seconds", subscriber, retryAfter)
			time.Sleep(time.Duration(retryAfter) * time.Second)
			h.sendText(subscriber, text, sendOptions)
		} else if errors.As(err, &telebotError) {
			if telebotError.Code == 403 {
				log.Printf("%d: %s, will be removed", subscriber, telebotError.Description)
				h.DB.Subscribers.Delete(subscriber)
			}
		} else {
			return err
		}
	}

	return nil
}
