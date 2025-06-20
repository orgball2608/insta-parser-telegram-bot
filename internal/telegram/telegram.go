package telegram

import (
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

type Client interface {
	SendFileToChannel(data tgbotapi.RequestFileData, dataType int) error
	GetUpdatesChan(u tgbotapi.UpdateConfig) tgbotapi.UpdatesChannel
	SendMessageToUser(message string)
	SendMessageToChanel(msg string)
	SendMediaToChanelByUrl(url string)
	SendMessage(chatID int64, text string) (int, error)
	StopReceivingUpdates()
	SendMediaGroup(media []interface{}) error
}
