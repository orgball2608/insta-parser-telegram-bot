package telegram

import (
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"log"
)

type Bot struct {
	Api *tgbotapi.BotAPI
}

func NewBot(token string) (*Bot, error) {
	bot, err := tgbotapi.NewBotAPI(token)
	if err != nil {
		log.Printf("Error creating bot: %v", err)
		return nil, err
	}

	return &Bot{
		Api: bot,
	}, nil
}

func (bot *Bot) SendToChannel(channelName string, data tgbotapi.RequestFileData, dataType int, username string) error {
	log.Printf("Sending to channel: %s", channelName)

	bot.SendMessage("@"+channelName, "New stories from "+username)

	if dataType == 1 {
		_, err := bot.Api.Send(tgbotapi.NewPhotoToChannel("@"+channelName, data))

		if err != nil {
			log.Printf("Error sending photo to channel: %v", err)
			return err
		}
	} else {
		videoConfig := tgbotapi.NewVideo(0, data)
		videoConfig.ChannelUsername = "@" + channelName

		_, err := bot.Api.Send(videoConfig)

		if err != nil {
			log.Printf("Error sending video to channel: %v", err)
			return err
		}
	}
	return nil
}

func (bot *Bot) SendError(user int64, err string) {
	msg := tgbotapi.NewMessage(user, err)
	_, err2 := bot.Api.Send(msg)
	if err2 != nil {
		return
	}
}

func (bot *Bot) SendMessage(channelName string, msg string) {
	newMsg := tgbotapi.NewMessageToChannel(channelName, msg)
	_, err := bot.Api.Send(newMsg)
	if err != nil {
		return
	}
}
