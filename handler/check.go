package handler

import (
	"encoding/json"
	"errors"
	"fmt"
	"gopkg.in/telebot.v3"
	"html"
	"io"
	"log"
	"net/http"
	"strings"
	"time"
)

const ApiUrl string = "https://www.tagesschau.de/api2"

type TagesschauResponse struct {
	News []struct {
		ExternalId   string `json:"externalId"`
		Date         string `json:"date"`
		Title        string `json:"title"`
		Url          string `json:"detailsweb"`
		BreakingNews bool   `json:"breakingNews"`
		Content      []struct {
			Value string `json:"value"`
		}
	} `json:"news"`
}

func (h Handler) OnTimer() {
	if h.Config.Debug {
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

	if len(result.News) == 0 {
		if h.Config.Debug {
			log.Println("No news found")
		}
		return nil
	}

	breakingNews := result.News[0]

	if !breakingNews.BreakingNews {
		if h.Config.Debug {
			log.Println("Not a breaking news")
		}
		return nil
	}

	if breakingNews.Url == "" {
		return errors.New("invalid breaking news")
	}

	if h.Config.LastEntry == breakingNews.ExternalId {
		if h.Config.Debug {
			log.Println("Already notified of this breaking news")
		}
		return nil
	}

	log.Println("New breaking news found")
	h.Config.LastEntry = breakingNews.ExternalId

	text := "<b>" + html.EscapeString(breakingNews.Title) + "</b>\n"

	time, _ := time.Parse("2006-01-02T15:04:05.000-07:00", breakingNews.Date)
	text += "<i>" + time.Format("02.01.2006 um 15:04:05 Uhr") + "</i>\n"

	text += ""
	if len(breakingNews.Content) > 0 && breakingNews.Content[0].Value != "" {
		content := breakingNews.Content[0].Value
		content = strings.Replace(content, "<em>", "", -1)
		content = strings.Replace(content, "</em>", "", -1)
		text += html.EscapeString(strings.TrimSpace(content)) + "\n"
	}

	postLink := strings.Replace(breakingNews.Url, "http://", "https://", -1)
	textLink := "<a href=\"" + postLink + "\">Eilmeldung aufrufen</a>"
	replyMarkup := h.Bot.NewMarkup()
	btn := replyMarkup.URL("Eilmeldung aufrufen", breakingNews.Url)
	replyMarkup.Inline(replyMarkup.Row(btn))

	for _, subscriber := range h.Config.Subscribers {
		if subscriber < 0 { // Group
			_, err = h.Bot.Send(telebot.ChatID(subscriber), "#EIL: "+text, &telebot.SendOptions{
				DisableWebPagePreview: true,
				ParseMode:             telebot.ModeHTML,
				ReplyMarkup:           replyMarkup,
			})
		} else {
			_, err = h.Bot.Send(telebot.ChatID(subscriber), text+textLink, defaultSendOptions)
		}

		if err != nil {
			if errors.Is(err, telebot.ErrChatNotFound) {
				log.Printf("Chat %d not found, will be deleted", subscriber)
				h.Config.RemoveSubscriber(subscriber)
			} else if errors.Is(err, telebot.ErrGroupMigrated) {
				log.Printf("Chat %d migrated to new group", subscriber)
				h.Config.RemoveSubscriber(subscriber)
				h.Config.AddSubscriber(err.(*telebot.GroupError).MigratedTo)
			} else {
				log.Println(err)
			}

		}
	}

	err = h.Config.Save()
	if err != nil {
		return fmt.Errorf("failed writing config: %w", err)
	}

	return nil
}
