package commandimpl

import (
	"context"
	"fmt"
	"strings"
	"time"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

func (c *CommandImpl) handleReelCommand(ctx context.Context, update tgbotapi.Update) error {
	reelURL := strings.TrimSpace(update.Message.CommandArguments())
	chatID := update.Message.Chat.ID

	if reelURL == "" {
		_, err := c.Telegram.SendMessage(chatID, "Please provide a Reel URL: /reel <instagram_reel_url>")
		return err
	}

	sentMsgID, err := c.Telegram.SendMessage(chatID, fmt.Sprintf("Fetching Reel from URL: %s... ⏳", reelURL))
	if err != nil {
		return fmt.Errorf("failed to send initial message: %w", err)
	}

	ctxWithTimeout, cancel := context.WithTimeout(ctx, 5*time.Minute)
	defer cancel()

	reel, err := c.Instagram.GetUserReel(ctxWithTimeout, reelURL)
	if err != nil {
		c.Telegram.EditMessageText(chatID, sentMsgID, fmt.Sprintf("❌ Error fetching Reel: %v", err))
		return fmt.Errorf("failed to get Reel from URL: %w", err)
	}

	if len(reel.MediaURLs) == 0 {
		c.Telegram.EditMessageText(chatID, sentMsgID, "Could not find any media in the provided URL.")
		return nil
	}

	c.Telegram.EditMessageText(chatID, sentMsgID, "✅ Successfully fetched Reel info! Sending video now...")

	var captionToSend string
	if reel.Caption != "" {
		if reel.Username != "" {
			captionToSend = fmt.Sprintf("Reel by @%s:\n\n%s", reel.Username, reel.Caption)
		} else {
			captionToSend = reel.Caption
		}
	}

	err = c.Telegram.SendMediaByUrl(chatID, reel.MediaURLs[0])
	if err != nil {
		c.Logger.Error("Failed to send Reel video", "error", err)
	}

	if captionToSend != "" {
		c.Telegram.SendMessage(chatID, captionToSend)
	}

	return nil
}
