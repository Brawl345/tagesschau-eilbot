package handler

import (
	"log"
	"os"

	"gopkg.in/telebot.v3"
)

var defaultSendOptions = &telebot.SendOptions{
	AllowWithoutReply:     true,
	DisableWebPagePreview: true,
	ParseMode:             telebot.ModeHTML,
}

func isDebugMode() bool {
	_, exists := os.LookupEnv("DEBUG")
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
