package telegramimpl

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/orgball2608/insta-parser-telegram-bot/pkg/logger"
	"github.com/orgball2608/insta-parser-telegram-bot/pkg/retry"
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

// SendMediaToChanelByUrl downloads and sends media from a URL to the channel.
func (tg *TelegramImpl) SendMediaToChanelByUrl(url string) {
	var media []byte
	var err error

	operation := func() error {
		media, err = tg.downloadMedia(url)
		return err
	}

	err = retry.Do(context.Background(), tg.Logger, "DownloadMedia", operation, retry.DefaultConfig())
	if err != nil {
		tg.Logger.Error("Failed to download media after several retries", "url", url, "error", err)
		return
	}

	if len(media) == 0 {
		tg.Logger.Error("Received empty media data after download", "url", url)
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

func (tg *TelegramImpl) downloadMedia(url string) ([]byte, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("error creating HTTP request: %w", err)
	}

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("http client error: %w", err)
	}
	defer safeClose(resp.Body, tg.Logger)

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("bad status code: %d", resp.StatusCode)
	}

	media, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("error reading media data: %w", err)
	}

	return media, nil
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
func (tg *TelegramImpl) GetUpdatesChan(u tgbotapi.UpdateConfig) tgbotapi.UpdatesChannel {
	return tg.TgBot.GetUpdatesChan(u)
}

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

func (tg *TelegramImpl) StopReceivingUpdates() {
	tg.TgBot.StopReceivingUpdates()
}

// SendMediaGroup sends a group of photos or videos as an album.
func (tg *TelegramImpl) SendMediaGroup(media []interface{}) error {
	if len(media) == 0 {
		return nil
	}
	if len(media) > 10 {
		tg.Logger.Warn("Attempted to send more than 10 media items in a group", "count", len(media))
		media = media[:10]
	}

	channelName := "@" + tg.Config.Telegram.Channel
	tg.Logger.Info("Sending media group to channel", "channel", channelName, "count", len(media))

	msg := tgbotapi.NewMediaGroup(0, media)
	msg.ChannelUsername = channelName

	_, err := tg.TgBot.Request(msg)
	if err != nil {
		tg.Logger.Error("Error sending media group via bot.Request", "channel", channelName, "error", err)
		return fmt.Errorf("failed to send media group: %w", err)
	}

	tg.Logger.Info("Successfully sent media group", "channel", channelName)
	return nil
}
