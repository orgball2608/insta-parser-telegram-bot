package telegram

import (
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

type Client interface {
	SendToChannel(data tgbotapi.RequestFileData, dataType int, username string) error
	SendMessageToUser(message string)
	SendMessageToChanel(msg string)
}
