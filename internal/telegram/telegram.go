package telegram

import (
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

type Client interface {
	GetUpdatesChan(u tgbotapi.UpdateConfig) tgbotapi.UpdatesChannel
	StopReceivingUpdates()

	SendMessage(chatID int64, text string) (int, error)
	SendMessageWithParseMode(chatID int64, text string, parseMode string) (int, error)
	SendMediaByUrl(chatID int64, url string) error
	SendMediaGroup(chatID int64, media []interface{}) error
	EditMessageText(chatID int64, messageID int, newText string) error
	DeleteMessage(config tgbotapi.DeleteMessageConfig) error
	Send(c tgbotapi.Chattable) (tgbotapi.Message, error)
	Request(c tgbotapi.Chattable) (*tgbotapi.APIResponse, error)

	SendMessageToDefaultChannel(msg string)
	SendMediaToDefaultChannelByUrl(url string)

	DownloadMedia(url string) ([]byte, error)
	DownloadMediaToTempFile(url string) (string, error)
}
