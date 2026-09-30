package handler

import (
	"context"
	"encoding/json"
	"fmt"
	"html"
	"io"
	"log"
	"net/http"
	"net/url"
	"slices"
	"strings"
	"time"

	"gopkg.in/telebot.v3"
)

const (
	apiUrl        = "https://www.tagesschau.de/json/headerapp"
	baseUrl       = "https://www.tagesschau.de/"
	maxBodySize   = 5 << 20
	firstCheck    = 5 * time.Second
	checkInterval = time.Minute
)

type breakingNews struct {
	Id       string `json:"id"`
	Headline string `json:"headline"`
	Text     string `json:"text"`
	Url      string `json:"url"`
	Date     string `json:"date"`
}

type tagesschauResponse struct {
	BreakingNews []breakingNews `json:"breakingNews"`
}

// Poll checks for breaking news until ctx is cancelled. A running broadcast
// always completes so that no subscriber misses an alert that is already
// marked as sent.
func (h *Handler) Poll(ctx context.Context) {
	timer := time.NewTimer(firstCheck)
	defer timer.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-timer.C:
		}

		if isDebugMode() {
			log.Println("Checking for breaking news")
		}
		if err := h.check(ctx); err != nil {
			log.Println(err)
		}
		timer.Reset(checkInterval)
	}
}

func (h *Handler) fetch(ctx context.Context) ([]breakingNews, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, apiUrl, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Accept", "application/json")

	resp, err := h.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("got HTTP error %s", resp.Status)
	}

	var result tagesschauResponse
	if err := json.NewDecoder(io.LimitReader(resp.Body, maxBodySize)).Decode(&result); err != nil {
		return nil, fmt.Errorf("can not unmarshal JSON: %w", err)
	}

	news := make([]breakingNews, 0, len(result.BreakingNews))
	for _, n := range result.BreakingNews {
		if n.Id == "" {
			continue
		}
		if _, err := articleUrl(n.Url); err != nil {
			log.Printf("Skipping breaking news %s: %s", n.Id, err)
			continue
		}
		news = append(news, n)
	}
	return news, nil
}

func (h *Handler) check(ctx context.Context) error {
	news, err := h.fetch(ctx)
	if err != nil {
		return err
	}

	if len(news) == 0 {
		if isDebugMode() {
			log.Println("No breaking news found")
		}
		return nil
	}

	empty, err := h.DB.SentNews.IsEmpty(ctx)
	if err != nil {
		return fmt.Errorf("error while reading sent news: %w", err)
	}
	if empty {
		for _, n := range news {
			if _, err := h.DB.SentNews.MarkSent(ctx, n.Id); err != nil {
				return fmt.Errorf("error while marking news as sent: %w", err)
			}
		}
		log.Printf("Initialized with %d current breaking news", len(news))
		return nil
	}

	// The API lists the newest alert first.
	for _, n := range slices.Backward(news) {
		isNew, err := h.DB.SentNews.MarkSent(ctx, n.Id)
		if err != nil {
			return fmt.Errorf("error while marking news as sent: %w", err)
		}
		if !isNew {
			if isDebugMode() {
				log.Printf("Already notified of breaking news %s", n.Id)
			}
			continue
		}

		log.Printf("New breaking news found: %s", n.Id)
		if err := h.broadcast(context.WithoutCancel(ctx), n); err != nil {
			return err
		}
	}

	if err := h.DB.SentNews.Prune(ctx); err != nil {
		return fmt.Errorf("error while pruning sent news: %w", err)
	}
	return nil
}

func articleUrl(raw string) (string, error) {
	base, _ := url.Parse(baseUrl)
	ref, err := url.Parse(strings.TrimSpace(raw))
	if err != nil {
		return "", err
	}
	u := base.ResolveReference(ref)
	if (u.Scheme != "https" && u.Scheme != "http") || u.Host == "" || raw == "" {
		return "", fmt.Errorf("invalid URL %q", raw)
	}
	return u.String(), nil
}

func (h *Handler) broadcast(ctx context.Context, n breakingNews) error {
	link, err := articleUrl(n.Url)
	if err != nil {
		return err
	}

	sb := strings.Builder{}
	fmt.Fprintf(&sb, "<b>%s</b>\n", html.EscapeString(strings.TrimSpace(n.Headline)))
	fmt.Fprintf(&sb, "<i>%s</i>\n", html.EscapeString(strings.TrimSpace(strings.Replace(n.Date, "Stand: ", "", 1))))
	if text := strings.TrimSpace(n.Text); text != "" {
		fmt.Fprintf(&sb, "%s\n", html.EscapeString(text))
	}

	groupText := "#EIL: " + sb.String()
	privateText := sb.String() + fmt.Sprintf("<a href=\"%s\">Eilmeldung aufrufen</a>", html.EscapeString(link))

	replyMarkup := h.Bot.NewMarkup()
	replyMarkup.Inline(replyMarkup.Row(replyMarkup.URL("Eilmeldung aufrufen", link)))
	groupOptions := &telebot.SendOptions{
		DisableWebPagePreview: true,
		ParseMode:             telebot.ModeHTML,
		ReplyMarkup:           replyMarkup,
	}

	subscribers, err := h.DB.Subscribers.GetAll(ctx)
	if err != nil {
		return fmt.Errorf("error while getting subscribers: %w", err)
	}

	h.sendAll(ctx, subscribers, func(chatId int64) (string, *telebot.SendOptions) {
		if chatId < 0 {
			return groupText, groupOptions
		}
		return privateText, defaultSendOptions
	})

	log.Printf("Sent breaking news %s to %d subscriber(s)", n.Id, len(subscribers))
	return nil
}
