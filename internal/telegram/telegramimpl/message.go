package telegramimpl

import (
	"context"
	"fmt"
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/orgball2608/insta-parser-telegram-bot/pkg/logger"
	"io"
	"net/http"
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

// SendImageToChanelByUrl downloads and sends an image from URL to the channel
func (tg *TelegramImpl) SendImageToChanelByUrl(url string) {
	// Create context with timeout for the HTTP request
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Create HTTP request with context
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		tg.Logger.Error("Error creating HTTP request", "url", url, "error", err)
		return
	}

	// Execute the request
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		tg.Logger.Error("Error downloading image", "url", url, "error", err)
		return
	}
	defer safeClose(resp.Body, tg.Logger)

	// Read image data
	media, err := io.ReadAll(resp.Body)
	if err != nil {
		tg.Logger.Error("Error reading image data", "url", url, "error", err)
		return
	}

	// Check if we actually got image data
	if len(media) == 0 {
		tg.Logger.Error("Received empty image data", "url", url)
		return
	}

	// Send the downloaded image to channel
	mediaBytes := tgbotapi.FileBytes{
		Name:  "media",
		Bytes: media,
	}

	if err := tg.SendFileToChannel(mediaBytes, 1); err != nil {
		tg.Logger.Error("Error sending image to channel", "url", url, "error", err)
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
