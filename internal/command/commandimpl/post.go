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

	var captionToSend string
	if post.Caption != "" {
		if post.Username != "" {
			captionToSend = fmt.Sprintf("From @%s:\n\n%s", post.Username, post.Caption)
		} else {
			captionToSend = post.Caption
		}
		c.Telegram.SendMessageToChanel(captionToSend)
	}

	for _, mediaURL := range post.MediaURLs {
		c.Telegram.SendMediaToChanelByUrl(mediaURL)
	}

	_, err = c.Telegram.SendMessage(update.Message.Chat.ID,
		fmt.Sprintf("Successfully sent %d media item(s) from the post.", len(post.MediaURLs)))

	return err
}
