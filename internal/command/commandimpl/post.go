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

func (c *CommandImpl) handlePostCommand(ctx context.Context, update tgbotapi.Update) error {
	postURL := strings.TrimSpace(update.Message.CommandArguments())
	chatID := update.Message.Chat.ID

	if postURL == "" {
		_, err := c.Telegram.SendMessage(chatID, "Please provide a post URL: /post <instagram_post_url>")
		return err
	}

	// Escape URL for Markdown
	escapedURL := formatter.EscapeMarkdownV2(postURL)
	initialMessage := fmt.Sprintf("Fetching post from URL: %s... ⏳", escapedURL)
	sentMsgID, err := c.Telegram.SendMessage(chatID, initialMessage)
	if err != nil {
		return fmt.Errorf("failed to send initial message: %w", err)
	}

	ctxWithTimeout, cancel := context.WithTimeout(ctx, 5*time.Minute)
	defer cancel()

	var post *domain.PostItem

	op := func() error {
		var opErr error
		post, opErr = c.Instagram.GetUserPost(ctxWithTimeout, postURL)
		return opErr
	}

	err = c.doWithRetryNotify(ctx, chatID, sentMsgID, initialMessage, "GetUserPost", op)

	if err != nil {
		c.Telegram.EditMessageText(chatID, sentMsgID, fmt.Sprintf("❌ Error fetching post: %v", err))
		return fmt.Errorf("failed to get post from URL: %w", err)
	}

	if len(post.MediaURLs) == 0 {
		c.Telegram.EditMessageText(chatID, sentMsgID, "Could not find any media in the provided URL.")
		return nil
	}

	c.Telegram.EditMessageText(chatID, sentMsgID, "✅ Successfully fetched post info! Sending media now...")

	mediaGroup := make([]interface{}, 0, len(post.MediaURLs))
	var captionBuilder strings.Builder
	if post.Username != "" {
		// Escape username for Markdown
		escapedUsername := formatter.EscapeMarkdownV2(post.Username)
		captionBuilder.WriteString(fmt.Sprintf("*Post by @%s*\n\n", escapedUsername))
	}
	if post.Caption != "" {
		// Escape caption for Markdown
		escapedCaption := formatter.EscapeMarkdownV2(post.Caption)
		captionBuilder.WriteString(escapedCaption)
		captionBuilder.WriteString("\n\n")
	}
	if post.LikeCount > 0 {
		captionBuilder.WriteString(fmt.Sprintf("❤️ %s", formatter.FormatNumber(post.LikeCount)))
	}
	if post.PostedAgo != "" {
		// Escape posted ago for Markdown
		escapedPostedAgo := formatter.EscapeMarkdownV2(post.PostedAgo)
		captionBuilder.WriteString(fmt.Sprintf(" | 🕒 %s\n", escapedPostedAgo))
	} else if post.LikeCount > 0 {
		captionBuilder.WriteString("\n")
	}

	// Escape post URL for Markdown
	escapedPostURL := formatter.EscapeMarkdownV2(post.PostURL)
	captionBuilder.WriteString(fmt.Sprintf("\n[View on Instagram](%s)", escapedPostURL))

	captionToSend := captionBuilder.String()

	for i, mediaURL := range post.MediaURLs {
		var mediaItem tgbotapi.RequestFileData = tgbotapi.FileURL(mediaURL)
		var media tgbotapi.BaseInputMedia

		if strings.Contains(mediaURL, ".mp4") {
			video := tgbotapi.NewInputMediaVideo(mediaItem)
			if i == 0 {
				video.Caption = captionToSend
			}
			media = video.BaseInputMedia
		} else {
			photo := tgbotapi.NewInputMediaPhoto(mediaItem)
			if i == 0 {
				photo.Caption = captionToSend
			}
			media = photo.BaseInputMedia
		}
		mediaGroup = append(mediaGroup, media)
	}

	if len(mediaGroup) > 0 {
		if err := c.Telegram.SendMediaGroup(chatID, mediaGroup); err != nil {
			c.Logger.Error("Failed to send media group, falling back to individual sending", "error", err)

			if captionToSend != "" {
				c.Telegram.SendMessage(chatID, captionToSend)
			}
			for _, mediaURL := range post.MediaURLs {
				c.Telegram.SendMediaByUrl(chatID, mediaURL)
			}
		}
	}

	return nil
}
