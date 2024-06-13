package telegramimpl

import (
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

func (t *TelegramImpl) SendFileToChannel(data tgbotapi.RequestFileData, dataType int) error {
	t.logger.Info("Sending to channel")
	chanelName := "@" + t.config.Telegram.Channel

	if dataType == 1 {
		_, err := t.tgBot.Send(tgbotapi.NewPhotoToChannel(chanelName, data))

		if err != nil {
			t.logger.Error("Error sending photo to channel", "Error", err)
			return err
		}
	} else {
		videoConfig := tgbotapi.NewVideo(0, data)
		videoConfig.ChannelUsername = chanelName

		_, err := t.tgBot.Send(videoConfig)

		if err != nil {
			t.logger.Error("Error sending video to channel", "Error", err)
			return err
		}
	}
	return nil
}

func (t *TelegramImpl) SendMessageToUser(message string) {
	msg := tgbotapi.NewMessage(t.config.Telegram.User, message)
	_, err := t.tgBot.Send(msg)
	if err != nil {
		t.logger.Error("Error sending message to user", "Error", err)
		return
	}
}

func (t *TelegramImpl) SendMessageToChanel(msg string) {
	newMsg := tgbotapi.NewMessageToChannel("@"+t.config.Telegram.Channel, msg)
	_, err := t.tgBot.Send(newMsg)
	if err != nil {
		t.logger.Error("Error sending message to channel", "Error", err)
		return
	}
}

func (t *TelegramImpl) GetUpdatesChan(u tgbotapi.UpdateConfig) (tgbotapi.UpdatesChannel, error) {
	return t.tgBot.GetUpdatesChan(u), nil
}
