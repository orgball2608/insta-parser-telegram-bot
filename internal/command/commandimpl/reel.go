package commandimpl

import (
	"context"
	"fmt"
	"strings"
	"time"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/orgball2608/insta-parser-telegram-bot/internal/domain"
	"github.com/orgball2608/insta-parser-telegram-bot/pkg/formatter"
)

func (c *CommandImpl) handleReelCommand(ctx context.Context, update tgbotapi.Update) error {
	reelURL := strings.TrimSpace(update.Message.CommandArguments())
	chatID := update.Message.Chat.ID

	if reelURL == "" {
		_, err := c.Telegram.SendMessage(chatID, "Please provide a Reel URL: /reel <instagram_reel_url>")
		return err
	}

	// Escape URL for Markdown
	escapedURL := formatter.EscapeMarkdownV2(reelURL)
	initialMessage := fmt.Sprintf("Fetching Reel from URL: %s... ⏳", escapedURL)
	sentMsgID, err := c.Telegram.SendMessage(chatID, initialMessage)
	if err != nil {
		return fmt.Errorf("failed to send initial message: %w", err)
	}

	ctxWithTimeout, cancel := context.WithTimeout(ctx, 5*time.Minute)
	defer cancel()

	var reel *domain.PostItem

	op := func() error {
		var opErr error
		reel, opErr = c.Instagram.GetUserReel(ctxWithTimeout, reelURL)
		return opErr
	}

	err = c.doWithRetryNotify(ctx, chatID, sentMsgID, initialMessage, "GetUserReel", op)

	if err != nil {
		c.Telegram.EditMessageText(chatID, sentMsgID, fmt.Sprintf("❌ Error fetching Reel: %v", err))
		return fmt.Errorf("failed to get Reel from URL: %w", err)
	}

	if len(reel.MediaURLs) == 0 {
		c.Telegram.EditMessageText(chatID, sentMsgID, "Could not find any media in the provided URL.")
		return nil
	}

	c.Telegram.EditMessageText(chatID, sentMsgID, "✅ Successfully fetched Reel info! Sending video now...")

	var captionBuilder strings.Builder
	if reel.Username != "" {
		// Escape username for Markdown
		escapedUsername := formatter.EscapeMarkdownV2(reel.Username)
		captionBuilder.WriteString(fmt.Sprintf("*Reel by @%s*\n\n", escapedUsername))
	}
	if reel.Caption != "" {
		// Escape caption for Markdown
		escapedCaption := formatter.EscapeMarkdownV2(reel.Caption)
		captionBuilder.WriteString(escapedCaption)
		captionBuilder.WriteString("\n\n")
	}
	if reel.LikeCount > 0 {
		captionBuilder.WriteString(fmt.Sprintf("❤️ %s", formatter.FormatNumber(reel.LikeCount)))
	}
	if reel.PostedAgo != "" {
		// Escape posted ago for Markdown
		escapedPostedAgo := formatter.EscapeMarkdownV2(reel.PostedAgo)
		captionBuilder.WriteString(fmt.Sprintf(" | 🕒 %s\n", escapedPostedAgo))
	} else if reel.LikeCount > 0 {
		captionBuilder.WriteString("\n")
	}

	// Escape post URL for Markdown
	escapedPostURL := formatter.EscapeMarkdownV2(reel.PostURL)
	captionBuilder.WriteString(fmt.Sprintf("\n[View on Instagram](%s)", escapedPostURL))

	captionToSend := captionBuilder.String()

	err = c.Telegram.SendMediaByUrl(chatID, reel.MediaURLs[0])
	if err != nil {
		c.Logger.Error("Failed to send Reel video", "error", err)
	}

	if captionToSend != "" {
		c.Telegram.SendMessage(chatID, captionToSend)
	}

	return nil
}
