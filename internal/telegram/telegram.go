package telegram

import (
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

type Client interface {
	GetUpdatesChan(u tgbotapi.UpdateConfig) tgbotapi.UpdatesChannel
	StopReceivingUpdates()

	SendMessage(chatID int64, text string) (int, error)
	SendMediaByUrl(chatID int64, url string) error
	SendMediaGroup(chatID int64, media []interface{}) error
	EditMessageText(chatID int64, messageID int, newText string) error

	SendMessageToDefaultChannel(msg string)
	SendMediaToDefaultChannelByUrl(url string)
}
