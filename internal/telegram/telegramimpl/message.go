package telegramimpl

import (
	"context"
	"fmt"
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/orgball2608/insta-parser-telegram-bot/pkg/logger"
	"io"
	"net/http"
	"strings"
	"time"
)

// SendFileToChannel sends a photo or video to the configured Telegram channel
func (tg *TelegramImpl) SendFileToChannel(data tgbotapi.RequestFileData, dataType int) error {
	channelName := "@" + tg.Config.Telegram.Channel
	tg.Logger.Info("Sending media to channel", "channel", channelName, "type", dataType)

	var err error
	switch dataType {
	case 1: // Photo
		photoMsg := tgbotapi.NewPhotoToChannel(channelName, data)
		_, err = tg.TgBot.Send(photoMsg)
	case 2: // Video
		videoConfig := tgbotapi.NewVideo(0, data)
		videoConfig.ChannelUsername = channelName
		_, err = tg.TgBot.Send(videoConfig)
	default:
		return fmt.Errorf("unsupported media type: %d", dataType)
	}

	if err != nil {
		tg.Logger.Error("Error sending media to channel",
			"channel", channelName,
			"type", dataType,
			"error", err)
		return fmt.Errorf("failed to send %s to channel: %w",
			getMediaTypeName(dataType), err)
	}

	tg.Logger.Info("Successfully sent media to channel",
		"channel", channelName,
		"type", getMediaTypeName(dataType))
	return nil
}

// SendMessageToUser sends a text message to the configured user
func (tg *TelegramImpl) SendMessageToUser(message string) {
	msg := tgbotapi.NewMessage(tg.Config.Telegram.User, message)
	_, err := tg.TgBot.Send(msg)
	if err != nil {
		tg.Logger.Error("Error sending message to user",
			"userID", tg.Config.Telegram.User,
			"error", err)
		return
	}

	tg.Logger.Info("Message sent to user",
		"userID", tg.Config.Telegram.User)
}

// SendMessageToChanel sends a text message to the configured channel
func (tg *TelegramImpl) SendMessageToChanel(msg string) {
	channelName := "@" + tg.Config.Telegram.Channel
	newMsg := tgbotapi.NewMessageToChannel(channelName, msg)

	_, err := tg.TgBot.Send(newMsg)
	if err != nil {
		tg.Logger.Error("Error sending message to channel",
			"channel", channelName,
			"error", err)
		return
	}

	tg.Logger.Info("Message sent to channel",
		"channel", channelName)
}

// SendMediaToChanelByUrl downloads and sends an media from URL to the channel
func (tg *TelegramImpl) SendMediaToChanelByUrl(url string) {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		tg.Logger.Error("Error creating HTTP request", "url", url, "error", err)
		return
	}

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		tg.Logger.Error("Error downloading media", "url", url, "error", err)
		return
	}
	defer safeClose(resp.Body, tg.Logger)

	media, err := io.ReadAll(resp.Body)
	if err != nil {
		tg.Logger.Error("Error reading media data", "url", url, "error", err)
		return
	}

	if len(media) == 0 {
		tg.Logger.Error("Received empty media data", "url", url)
		return
	}

	mediaType := 1
	if strings.Contains(url, ".mp4") {
		mediaType = 2
	}

	mediaBytes := tgbotapi.FileBytes{
		Name:  "media",
		Bytes: media,
	}

	if err := tg.SendFileToChannel(mediaBytes, mediaType); err != nil {
		tg.Logger.Error("Error sending media to channel", "url", url, "type", getMediaTypeName(mediaType), "error", err)
	}
}

// SendMessage sends a message to a specific chat ID
func (tg *TelegramImpl) SendMessage(chatID int64, text string) (int, error) {
	msg := tgbotapi.NewMessage(chatID, text)
	sentMsg, err := tg.TgBot.Send(msg)
	if err != nil {
		tg.Logger.Error("Error sending message",
			"chatID", chatID,
			"error", err)
		return 0, fmt.Errorf("failed to send message: %w", err)
	}

	tg.Logger.Info("Message sent",
		"chatID", chatID,
		"messageID", sentMsg.MessageID)
	return sentMsg.MessageID, nil
}

// GetUpdatesChan wraps the bot's GetUpdatesChan method
func (tg *TelegramImpl) GetUpdatesChan(u tgbotapi.UpdateConfig) (tgbotapi.UpdatesChannel, error) {
	return tg.TgBot.GetUpdatesChan(u), nil
}

// Helper functions

// getMediaTypeName returns the string name of a media type
func getMediaTypeName(dataType int) string {
	switch dataType {
	case 1:
		return "photo"
	case 2:
		return "video"
	default:
		return "unknown media"
	}
}

// safeClose safely closes an io.ReadCloser and logs any errors
func safeClose(closer io.ReadCloser, logger logger.Logger) {
	if err := closer.Close(); err != nil {
		logger.Error("Error closing response body", "error", err)
	}
}
