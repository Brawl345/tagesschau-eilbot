package handler

import (
	"gopkg.in/telebot.v3"
	"log"
	"os"
)

var defaultSendOptions = &telebot.SendOptions{
	AllowWithoutReply:     true,
	DisableWebPagePreview: true,
	ParseMode:             telebot.ModeHTML,
}

func isDebugMode() bool {
	_, exists := os.LookupEnv("TAGESSCHAU_EILBOT_DEBUG")
	return exists
}

func isGroupAdmin(bot *telebot.Bot, chatId, userId int64) bool {
	data, err := bot.ChatMemberOf(telebot.ChatID(chatId), telebot.ChatID(userId))

	if err != nil {
		log.Println("is_group_admin() errored:", err)
		return false
	}

	return data.Role == telebot.Creator || data.Role == telebot.Administrator
}
