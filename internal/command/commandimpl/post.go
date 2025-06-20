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

	if postURL == "" {
		_, err := c.Telegram.SendMessage(update.Message.Chat.ID,
			"Please provide a post URL: /post <instagram_post_url>")
		return err
	}

	_, err := c.Telegram.SendMessage(update.Message.Chat.ID,
		fmt.Sprintf("Getting post from URL: %s...", postURL))
	if err != nil {
		return fmt.Errorf("failed to send initial message: %w", err)
	}

	ctxWithTimeout, cancel := context.WithTimeout(ctx, 5*time.Minute)
	defer cancel()

	post, err := c.Instagram.GetUserPost(ctxWithTimeout, postURL)
	if err != nil {
		return fmt.Errorf("failed to get post from URL: %w", err)
	}

	if len(post.MediaURLs) == 0 {
		_, err := c.Telegram.SendMessage(update.Message.Chat.ID, "Could not find any media in the provided post URL.")
		return err
	}

	mediaGroup := make([]interface{}, 0, len(post.MediaURLs))

	for i, mediaURL := range post.MediaURLs {
		var mediaItem tgbotapi.RequestFileData = tgbotapi.FileURL(mediaURL)

		if strings.Contains(mediaURL, ".mp4") {
			video := tgbotapi.NewInputMediaVideo(mediaItem)
			if i == 0 && post.Caption != "" {
				if post.Username != "" {
					video.Caption = fmt.Sprintf("From @%s:\n\n%s", post.Username, post.Caption)
				} else {
					video.Caption = post.Caption
				}
			}
			mediaGroup = append(mediaGroup, video)
		} else {
			photo := tgbotapi.NewInputMediaPhoto(mediaItem)
			if i == 0 && post.Caption != "" {
				if post.Username != "" {
					photo.Caption = fmt.Sprintf("From @%s:\n\n%s", post.Username, post.Caption)
				} else {
					photo.Caption = post.Caption
				}
			}
			mediaGroup = append(mediaGroup, photo)
		}
	}

	if len(mediaGroup) > 0 {
		if err := c.Telegram.SendMediaGroup(mediaGroup); err != nil {
			c.Logger.Error("Failed to send media group, falling back to individual sending", "error", err)

			if post.Caption != "" {
				c.Telegram.SendMessageToChanel(fmt.Sprintf("From @%s:\n\n%s", post.Username, post.Caption))
			}
			for _, mediaURL := range post.MediaURLs {
				c.Telegram.SendMediaToChanelByUrl(mediaURL)
			}
		}
	}

	_, err = c.Telegram.SendMessage(update.Message.Chat.ID,
		fmt.Sprintf("Successfully sent %d media item(s) from the post.", len(post.MediaURLs)))

	return err
}
