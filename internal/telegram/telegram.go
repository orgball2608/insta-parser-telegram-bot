package telegram

import (
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

type Client interface {
	SendFileToChannel(data tgbotapi.RequestFileData, dataType int) error
	GetUpdatesChan(u tgbotapi.UpdateConfig) (tgbotapi.UpdatesChannel, error)
	SendMessageToUser(message string)
	SendMessageToChanel(msg string)
	SendImageToChanelByUrl(url string)
	// SendMessage sends a message to a specific chat ID and returns message ID and error
	SendMessage(chatID int64, text string) (int, error)
}
