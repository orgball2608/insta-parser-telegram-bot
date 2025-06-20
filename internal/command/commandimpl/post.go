package commandimpl

import (
	"context"
	"fmt"
	"strings"
	"time"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

func (c *CommandImpl) handlePostCommand(ctx context.Context, update tgbotapi.Update) error {
	postURL := strings.TrimSpace(update.Message.CommandArguments())
	chatID := update.Message.Chat.ID

	if postURL == "" {
		_, err := c.Telegram.SendMessage(chatID, "Please provide a post URL: /post <instagram_post_url>")
		return err
	}

	sentMsgID, err := c.Telegram.SendMessage(chatID, fmt.Sprintf("Fetching post from URL: %s... ⏳", postURL))
	if err != nil {
		return fmt.Errorf("failed to send initial message: %w", err)
	}

	ctxWithTimeout, cancel := context.WithTimeout(ctx, 5*time.Minute)
	defer cancel()

	post, err := c.Instagram.GetUserPost(ctxWithTimeout, postURL)
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
		captionBuilder.WriteString(fmt.Sprintf("*Post by @%s*\n\n", post.Username))
	}
	if post.Caption != "" {
		captionBuilder.WriteString(post.Caption)
		captionBuilder.WriteString("\n\n")
	}
	if post.LikeCount > 0 {
		captionBuilder.WriteString(fmt.Sprintf("❤️ %s", formatNumber(post.LikeCount)))
	}
	if post.PostedAgo != "" {
		captionBuilder.WriteString(fmt.Sprintf(" | 🕒 %s\n", post.PostedAgo))
	} else if post.LikeCount > 0 {
		captionBuilder.WriteString("\n")
	}

	captionBuilder.WriteString(fmt.Sprintf("\n[View on Instagram](%s)", post.PostURL))

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
