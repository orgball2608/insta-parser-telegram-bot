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

func (tg *TelegramImpl) GetUpdatesChan(u tgbotapi.UpdateConfig) tgbotapi.UpdatesChannel {
	return tg.TgBot.GetUpdatesChan(u)
}

func (tg *TelegramImpl) StopReceivingUpdates() {
	tg.TgBot.StopReceivingUpdates()
}

func (tg *TelegramImpl) SendMessage(chatID int64, text string) (int, error) {
	msg := tgbotapi.NewMessage(chatID, text)
	sentMsg, err := tg.TgBot.Send(msg)
	if err != nil {
		tg.Logger.Error("Error sending message", "chatID", chatID, "error", err)
		return 0, fmt.Errorf("failed to send message: %w", err)
	}
	tg.Logger.Info("Message sent", "chatID", chatID, "messageID", sentMsg.MessageID)
	return sentMsg.MessageID, nil
}

func (tg *TelegramImpl) SendMediaByUrl(chatID int64, url string) error {
	media, err := tg.downloadWithRetry(url)
	if err != nil {
		return err
	}
	return tg.sendMedia(chatID, media, url)
}

func (tg *TelegramImpl) SendMediaGroup(chatID int64, media []interface{}) error {
	if len(media) == 0 {
		return nil
	}
	if len(media) > 10 {
		tg.Logger.Warn("Cannot send more than 10 media items in a group, sending first 10.", "count", len(media))
		media = media[:10]
	}

	tg.Logger.Info("Sending media group", "chatID", chatID, "count", len(media))
	msg := tgbotapi.NewMediaGroup(chatID, media)

	_, err := tg.TgBot.Request(msg)
	if err != nil {
		tg.Logger.Error("Error sending media group via bot.Request", "chatID", chatID, "error", err)
		return fmt.Errorf("failed to send media group: %w", err)
	}

	tg.Logger.Info("Successfully sent media group", "chatID", chatID)
	return nil
}

func (tg *TelegramImpl) SendMessageToDefaultChannel(msg string) {
	if tg.Config.Telegram.Channel == "" {
		tg.Logger.Warn("Default channel not configured, skipping message.")
		return
	}
	channelName := "@" + tg.Config.Telegram.Channel
	newMsg := tgbotapi.NewMessageToChannel(channelName, msg)
	if _, err := tg.TgBot.Send(newMsg); err != nil {
		tg.Logger.Error("Error sending message to default channel", "channel", channelName, "error", err)
	} else {
		tg.Logger.Info("Message sent to default channel", "channel", channelName)
	}
}

func (tg *TelegramImpl) SendMediaToDefaultChannelByUrl(url string) {
	if tg.Config.Telegram.Channel == "" {
		tg.Logger.Warn("Default channel not configured, skipping media.")
		return
	}

	media, err := tg.downloadWithRetry(url)
	if err != nil {
		tg.SendMessageToDefaultChannel(fmt.Sprintf("Failed to download media: %s\nError: %v", url, err))
		return
	}

	channelName := "@" + tg.Config.Telegram.Channel
	if err := tg.sendMediaToChannel(channelName, media, url); err != nil {
		tg.Logger.Error("Failed sending media to default channel", "url", url, "error", err)
	}
}

func (tg *TelegramImpl) DownloadMedia(url string) ([]byte, error) {
	return tg.downloadWithRetry(url)
}

func (tg *TelegramImpl) downloadWithRetry(url string) ([]byte, error) {
	var media []byte
	var downloadErr error
	operation := func() error {
		media, downloadErr = tg.downloadMedia(url)
		return downloadErr
	}
	err := retry.Do(context.Background(), tg.Logger, "DownloadMedia", operation, retry.DefaultConfig())
	if err != nil {
		return nil, fmt.Errorf("failed to download media from %s after retries: %w", url, err)
	}
	if len(media) == 0 {
		return nil, fmt.Errorf("received empty media data from url: %s", url)
	}
	return media, nil
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
	return io.ReadAll(resp.Body)
}

func (tg *TelegramImpl) sendMedia(chatID int64, mediaBytes []byte, originalURL string) error {
	mediaType := 1 // Photo by default
	if strings.Contains(originalURL, ".mp4") {
		mediaType = 2
	}
	file := tgbotapi.FileBytes{Name: "media", Bytes: mediaBytes}

	var msg tgbotapi.Chattable
	switch mediaType {
	case 1:
		msg = tgbotapi.NewPhoto(chatID, file)
	case 2:
		msg = tgbotapi.NewVideo(chatID, file)
	default:
		return fmt.Errorf("unsupported media type")
	}

	if _, err := tg.TgBot.Send(msg); err != nil {
		tg.Logger.Error("Error sending file", "chatID", chatID, "error", err)
		return fmt.Errorf("failed to send file: %w", err)
	}
	return nil
}

func (tg *TelegramImpl) sendMediaToChannel(channelName string, mediaBytes []byte, originalURL string) error {
	mediaType := 1
	if strings.Contains(originalURL, ".mp4") {
		mediaType = 2
	}
	file := tgbotapi.FileBytes{Name: "media", Bytes: mediaBytes}

	var msg tgbotapi.Chattable
	switch mediaType {
	case 1:
		photoMsg := tgbotapi.NewPhotoToChannel(channelName, file)
		msg = photoMsg
	case 2:
		videoConfig := tgbotapi.NewVideo(0, file)
		videoConfig.ChannelUsername = channelName
		_, err := tg.TgBot.Send(videoConfig)
		if err != nil {
			tg.Logger.Error("Error sending video to channel", "channel", channelName, "error", err)
			return fmt.Errorf("failed to send video to channel: %w", err)
		}
		return nil
	default:
		return fmt.Errorf("unsupported media type")
	}

	if _, err := tg.TgBot.Send(msg); err != nil {
		tg.Logger.Error("Error sending file to channel", "channel", channelName, "error", err)
		return fmt.Errorf("failed to send file to channel: %w", err)
	}
	return nil
}

func safeClose(closer io.ReadCloser, logger logger.Logger) {
	if err := closer.Close(); err != nil {
		logger.Error("Error closing response body", "error", err)
	}
}

func (tg *TelegramImpl) EditMessageText(chatID int64, messageID int, newText string) error {
	editMsg := tgbotapi.NewEditMessageText(chatID, messageID, newText)
	editMsg.ParseMode = tgbotapi.ModeMarkdown

	_, err := tg.TgBot.Send(editMsg)
	if err != nil {
		tg.Logger.Error("Error editing message", "chatID", chatID, "messageID", messageID, "error", err)
		return fmt.Errorf("failed to edit message: %w", err)
	}
	tg.Logger.Info("Message edited successfully", "chatID", chatID, "messageID", messageID)
	return nil
}

func (tg *TelegramImpl) SendMessageWithParseMode(chatID int64, text string, parseMode string) (int, error) {
	msg := tgbotapi.NewMessage(chatID, text)
	msg.ParseMode = parseMode
	sentMsg, err := tg.TgBot.Send(msg)
	if err != nil {
		tg.Logger.Error("Error sending message with parse mode", "chatID", chatID, "error", err)
		return 0, fmt.Errorf("failed to send message: %w", err)
	}
	tg.Logger.Info("Message with parse mode sent", "chatID", chatID, "messageID", sentMsg.MessageID)
	return sentMsg.MessageID, nil
}

func (tg *TelegramImpl) DeleteMessage(config tgbotapi.DeleteMessageConfig) error {
	_, err := tg.TgBot.Request(config)
	if err != nil {
		tg.Logger.Error("Error deleting message", "chatID", config.ChatID, "messageID", config.MessageID, "error", err)
		return fmt.Errorf("failed to delete message: %w", err)
	}
	tg.Logger.Info("Message deleted successfully", "chatID", config.ChatID, "messageID", config.MessageID)
	return nil
}

func (tg *TelegramImpl) Send(c tgbotapi.Chattable) (tgbotapi.Message, error) {
	message, err := tg.TgBot.Send(c)
	if err != nil {
		tg.Logger.Error("Error sending message", "error", err)
		return tgbotapi.Message{}, fmt.Errorf("failed to send message: %w", err)
	}
	return message, nil
}
