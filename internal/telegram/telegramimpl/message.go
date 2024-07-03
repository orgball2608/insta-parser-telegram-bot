package telegramimpl

import (
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"io"
	"net/http"
)

func (tg *TelegramImpl) SendFileToChannel(data tgbotapi.RequestFileData, dataType int) error {
	tg.Logger.Info("Sending to channel")
	chanelName := "@" + tg.Config.Telegram.Channel

	if dataType == 1 {
		_, err := tg.TgBot.Send(tgbotapi.NewPhotoToChannel(chanelName, data))

		if err != nil {
			tg.Logger.Error("Error sending photo to channel", "Error", err)
			return err
		}
	} else {
		videoConfig := tgbotapi.NewVideo(0, data)
		videoConfig.ChannelUsername = chanelName

		_, err := tg.TgBot.Send(videoConfig)

		if err != nil {
			tg.Logger.Error("Error sending video to channel", "Error", err)
			return err
		}
	}
	return nil
}

func (tg *TelegramImpl) SendMessageToUser(message string) {
	msg := tgbotapi.NewMessage(tg.Config.Telegram.User, message)
	_, err := tg.TgBot.Send(msg)
	if err != nil {
		tg.Logger.Error("Error sending message to user", "Error", err)
		return
	}
}

func (tg *TelegramImpl) SendMessageToChanel(msg string) {
	newMsg := tgbotapi.NewMessageToChannel("@"+tg.Config.Telegram.Channel, msg)
	_, err := tg.TgBot.Send(newMsg)
	if err != nil {
		tg.Logger.Error("Error sending message to channel", "Error", err)
		return
	}
}

func (tg *TelegramImpl) SendImageToChanelByUrl(url string) {
	resp, err := http.Get(url)
	if err != nil {
		tg.Logger.Error("Error downloading image", "Error", err)
	}
	defer func(Body io.ReadCloser) {
		err := Body.Close()
		if err != nil {
			tg.Logger.Error("Error close http", "Error", err)
		}
	}(resp.Body)

	media, err := io.ReadAll(resp.Body)
	if err != nil {
		tg.Logger.Error("Error reading image data", "Error", err)
	}

	mediaBytes := tgbotapi.FileBytes{
		Name:  "media",
		Bytes: media,
	}

	err = tg.SendFileToChannel(mediaBytes, 1)
	if err != nil {
		tg.Logger.Error("Error sending message to channel", "Error", err)
		return
	}
}

func (tg *TelegramImpl) GetUpdatesChan(u tgbotapi.UpdateConfig) (tgbotapi.UpdatesChannel, error) {
	return tg.TgBot.GetUpdatesChan(u), nil
}
