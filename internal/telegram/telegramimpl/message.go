package telegramimpl

import (
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

func (t *TelegramImpl) SendToChannel(data tgbotapi.RequestFileData, dataType int, username string) error {
	t.logger.Info("Sending to channel")
	chanelName := "@" + t.config.Telegram.Channel

	t.SendMessageToChanel("New stories from " + username)

	if dataType == 1 {
		_, err := t.Api.Send(tgbotapi.NewPhotoToChannel(chanelName, data))

		if err != nil {
			t.logger.Error("Error sending photo to channel", "Error", err)
			return err
		}
	} else {
		videoConfig := tgbotapi.NewVideo(0, data)
		videoConfig.ChannelUsername = chanelName

		_, err := t.Api.Send(videoConfig)

		if err != nil {
			t.logger.Error("Error sending video to channel", "Error", err)
			return err
		}
	}
	return nil
}

func (t *TelegramImpl) SendMessageToUser(message string) {
	msg := tgbotapi.NewMessage(t.config.Telegram.User, message)
	_, err := t.Api.Send(msg)
	if err != nil {
		t.logger.Error("Error sending message to user", "Error", err)
		return
	}
}

func (t *TelegramImpl) SendMessageToChanel(msg string) {
	newMsg := tgbotapi.NewMessageToChannel(t.config.Telegram.Channel, msg)
	_, err := t.Api.Send(newMsg)
	if err != nil {
		t.logger.Error("Error sending message to channel", "Error", err)
		return
	}
}
