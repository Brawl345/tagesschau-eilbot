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

const ApiUrl string = "https://www.tagesschau.de/ipa/v1/web/headerapp/"

type TagesschauResponse struct {
	BreakingNews struct {
		Id       string `json:"id"`
		Headline string `json:"headline"`
		Text     string `json:"text"`
		Url      string `json:"url"`
		Date     string `json:"date"`
	} `json:"breakingNews"`
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

	if result.BreakingNews.Id == "" {
		if h.Config.Debug {
			log.Println("No breaking news found")
		}
		return nil
	}

	if result.BreakingNews.Url == "" {
		return errors.New("invalid breaking news")
	}

	if h.Config.LastEntry == result.BreakingNews.Id {
		if h.Config.Debug {
			log.Println("Already notified of this breaking news")
		}
		return nil
	}

	log.Println("New breaking news found")
	h.Config.LastEntry = result.BreakingNews.Id

	sb := strings.Builder{}

	sb.WriteString(fmt.Sprintf("<b>%s</b>\n", html.EscapeString(result.BreakingNews.Headline)))
	sb.WriteString(fmt.Sprintf("<i>%s</i>\n", html.EscapeString(result.BreakingNews.Date)))
	if result.BreakingNews.Text != "" {
		sb.WriteString(fmt.Sprintf("%s\n", html.EscapeString(strings.TrimSpace(result.BreakingNews.Text))))
	}

	textLink := fmt.Sprintf("<a href=\"%s\">Eilmeldung aufrufen</a>", result.BreakingNews.Url)
	replyMarkup := h.Bot.NewMarkup()
	btn := replyMarkup.URL("Eilmeldung aufrufen", result.BreakingNews.Url)
	replyMarkup.Inline(replyMarkup.Row(btn))

	groupText := "#EIL: " + sb.String()
	privateText := sb.String() + textLink

	for _, subscriber := range h.Config.Subscribers {
		if subscriber < 0 { // Group
			_, err = h.Bot.Send(telebot.ChatID(subscriber), groupText, &telebot.SendOptions{
				DisableWebPagePreview: true,
				ParseMode:             telebot.ModeHTML,
				ReplyMarkup:           replyMarkup,
			})
		} else {
			_, err = h.Bot.Send(telebot.ChatID(subscriber), privateText, defaultSendOptions)
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
