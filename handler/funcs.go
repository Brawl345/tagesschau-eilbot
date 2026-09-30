package handler

import (
	"log"
	"os"
	"strconv"
	"sync"

	"gopkg.in/telebot.v3"
)

var defaultSendOptions = &telebot.SendOptions{
	AllowWithoutReply:     true,
	DisableWebPagePreview: true,
	ParseMode:             telebot.ModeHTML,
}

var isDebugMode = sync.OnceValue(func() bool {
	debug, _ := strconv.ParseBool(os.Getenv("DEBUG"))
	return debug
})

// isAuthorized allows private chats, anonymous admins posting as the group
// itself and group members with the creator or administrator role.
func isAuthorized(c telebot.Context) bool {
	msg := c.Message()
	if msg == nil || msg.Private() {
		return msg != nil
	}

	if msg.SenderChat != nil && msg.SenderChat.ID == c.Chat().ID {
		return true
	}
	if msg.Sender == nil {
		return false
	}

	member, err := c.Bot().ChatMemberOf(c.Chat(), msg.Sender)
	if err != nil {
		log.Println("isAuthorized() errored:", err)
		return false
	}

	return member.Role == telebot.Creator || member.Role == telebot.Administrator
}
