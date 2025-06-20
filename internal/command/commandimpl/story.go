package commandimpl

import (
	"context"
	"errors"
	"fmt"
	"runtime/debug"
	"strings"
	"sync/atomic"
	"time"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/orgball2608/insta-parser-telegram-bot/internal/domain"
	"github.com/orgball2608/insta-parser-telegram-bot/internal/instagram"
)

const helpMessage = `👋 *Welcome to the Instagram Parser Bot!*

Here are the available commands:

*AUTOMATIC SUBSCRIPTIONS:*
/subscribe <username> - Subscribe to a user to get new stories automatically.
/unsubscribe <username> - Unsubscribe from a user.
/listsubscriptions - List all your current subscriptions.

*ONE-TIME DOWNLOADS:*
/story <username> - Fetch all current stories from a user.
/highlights <username> - Fetch all highlights from a user.
/post <post_url> - Download a post (photo/video/album) from its URL.
/reel <reel_url> - Download a Reel from its URL.

Type /help at any time to see this guide.`

func (c *CommandImpl) HandleCommand(ctx context.Context) error {
	u := tgbotapi.NewUpdate(0)
	u.Timeout = 60

	updates := c.Telegram.GetUpdatesChan(u)
	c.Logger.Info("Command handler started, listening for updates.")

	for {
		select {
		case <-ctx.Done():
			c.Logger.Info("Command handler shutting down.")
			c.Telegram.StopReceivingUpdates()
			return ctx.Err()
		case update, ok := <-updates:
			if !ok {
				c.Logger.Warn("Telegram updates channel closed unexpectedly. Restarting handler...")
				return errors.New("telegram updates channel closed")
			}

			go func(u tgbotapi.Update) {
				defer func() {
					if r := recover(); r != nil {
						c.Logger.Error("Panic recovered while processing an update", "panic", r, "stack", string(debug.Stack()))
					}
				}()

				if u.Message == nil {
					return
				}

				c.Logger.Info("Message received", "from", u.Message.From.UserName, "text", u.Message.Text)

				if u.Message.IsCommand() {
					if err := c.processCommand(ctx, u); err != nil {
						c.Logger.Error("Error processing command",
							"command", u.Message.Command(),
							"error", err)
					}
				}
			}(update)
		}
	}
}

func (c *CommandImpl) processCommand(ctx context.Context, update tgbotapi.Update) error {
	command := update.Message.Command()
	args := update.Message.CommandArguments()
	chatID := update.Message.Chat.ID

	switch command {
	case "start", "help":
		_, err := c.Telegram.SendMessage(chatID, helpMessage)
		return err
	case "subscribe":
		c.handleSubscribe(ctx, chatID, args)
		return nil
	case "unsubscribe":
		c.handleUnsubscribe(ctx, chatID, args)
		return nil
	case "listsubscriptions":
		c.handleListSubscriptions(ctx, chatID)
		return nil
	case "story":
		return c.handleStoryCommand(ctx, update)
	case "highlights":
		return c.handleHighlightsCommand(ctx, update)
	case "post":
		return c.handlePostCommand(ctx, update)
	case "reel":
		return c.handleReelCommand(ctx, update)
	default:
		_, err := c.Telegram.SendMessage(chatID, "Unknown command. Type /help to see the list of available commands.")
		return err
	}
}

func (c *CommandImpl) handleStoryCommand(ctx context.Context, update tgbotapi.Update) error {
	args := strings.TrimSpace(update.Message.CommandArguments())
	userName := strings.TrimSpace(args)
	chatID := update.Message.Chat.ID

	if userName == "" {
		_, err := c.Telegram.SendMessage(chatID, "Please provide a username: /story <username>")
		return err
	}

	initialMessage := fmt.Sprintf("Fetching stories for @%s... ⏳", userName)
	sentMsgID, err := c.Telegram.SendMessage(chatID, initialMessage)
	if err != nil {
		return fmt.Errorf("failed to send initial message: %w", err)
	}

	var stories []domain.StoryItem
	op := func() error {
		var opErr error
		stories, opErr = c.Instagram.GetUserStories(userName)
		return opErr
	}

	err = c.doWithRetryNotify(ctx, chatID, sentMsgID, initialMessage, "GetUserStories", op)
	if err != nil {
		errMsg := fmt.Sprintf("❌ Error fetching stories for @%s: %v", userName, err)
		if errors.Is(err, instagram.ErrPrivateAccount) {
			errMsg = fmt.Sprintf("Account @%s is private, I cannot fetch stories.", userName)
		}
		c.Telegram.EditMessageText(chatID, sentMsgID, errMsg)
		return err
	}

	if len(stories) == 0 {
		c.Telegram.EditMessageText(chatID, sentMsgID, fmt.Sprintf("No current stories found for @%s.", userName))
		return nil
	}

	c.Telegram.EditMessageText(chatID, sentMsgID, fmt.Sprintf("✅ Found %d stories for @%s. Sending now...", len(stories), userName))

	if err := c.Parser.ClearCurrentStories(userName); err != nil {
		c.Logger.Error("Error clearing current stories", "error", err)
	}

	for _, item := range stories {
		if item.MediaURL == "" {
			continue
		}
		if err := c.Telegram.SendMediaByUrl(chatID, item.MediaURL); err != nil {
			c.Logger.Error("Failed to send story media", "url", item.MediaURL, "error", err)
		}
	}

	c.Telegram.SendMessage(chatID, fmt.Sprintf("Finished sending %d stories for @%s.", len(stories), userName))
	return nil
}

func (c *CommandImpl) handleHighlightsCommand(ctx context.Context, update tgbotapi.Update) error {
	args := strings.TrimSpace(update.Message.CommandArguments())
	userName := strings.TrimSpace(args)
	chatID := update.Message.Chat.ID

	if userName == "" {
		_, err := c.Telegram.SendMessage(chatID, "Please provide a username: /highlights <username>")
		return err
	}

	initialMessage := fmt.Sprintf("Fetching highlights for @%s... This may take a while. ⏳", userName)
	sentMsgID, err := c.Telegram.SendMessage(chatID, initialMessage)
	if err != nil {
		return fmt.Errorf("failed to send initial message: %w", err)
	}

	var processedCount int64
	var reelsFound bool
	var totalItems int64

	processor := func(highlightReel domain.HighlightReel) error {
		reelsFound = true
		totalItems += int64(len(highlightReel.Items))

		c.Telegram.EditMessageText(chatID, sentMsgID, fmt.Sprintf("Processing album '%s' (%d items)... ⏳", highlightReel.Title, len(highlightReel.Items)))

		if len(highlightReel.Items) == 0 {
			return nil
		}

		mediaGroup := make([]interface{}, 0, len(highlightReel.Items))
		caption := fmt.Sprintf("Highlight: %s", highlightReel.Title)

		for i, item := range highlightReel.Items {
			if item.MediaURL == "" {
				continue
			}

			highlightItem := domain.Highlights{
				UserName:  userName,
				MediaURL:  item.MediaURL,
				CreatedAt: time.Now(),
			}
			if err := c.Parser.SaveHighlight(highlightItem); err != nil {
				c.Logger.Error("Error saving highlight to DB", "url", item.MediaURL, "error", err)
			}

			var mediaItem tgbotapi.RequestFileData = tgbotapi.FileURL(item.MediaURL)
			if strings.Contains(item.MediaURL, ".mp4") {
				video := tgbotapi.NewInputMediaVideo(mediaItem)
				if i == 0 {
					video.Caption = caption
				}
				mediaGroup = append(mediaGroup, video)
			} else {
				photo := tgbotapi.NewInputMediaPhoto(mediaItem)
				if i == 0 {
					photo.Caption = caption
				}
				mediaGroup = append(mediaGroup, photo)
			}
		}

		if len(mediaGroup) > 0 {
			if err := c.Telegram.SendMediaGroup(chatID, mediaGroup); err != nil {
				c.Logger.Error("Failed to send highlight media group, falling back", "title", highlightReel.Title, "error", err)
				c.Telegram.SendMessage(chatID, caption)
				for _, item := range highlightReel.Items {
					if item.MediaURL != "" {
						c.Telegram.SendMediaByUrl(chatID, item.MediaURL)
					}
				}
			}
		}
		atomic.AddInt64(&processedCount, int64(len(highlightReel.Items)))
		return nil
	}

	op := func() error {
		return c.Instagram.GetUserHighlights(userName, processor)
	}

	err = c.doWithRetryNotify(ctx, chatID, sentMsgID, initialMessage, "GetUserHighlights", op)
	if err != nil {
		errMsg := fmt.Sprintf("❌ Error fetching highlights for @%s: %v", userName, err)
		if errors.Is(err, instagram.ErrPrivateAccount) {
			errMsg = fmt.Sprintf("Account @%s is private, I cannot fetch highlights.", userName)
		}
		c.Telegram.EditMessageText(chatID, sentMsgID, errMsg)
		return err
	}

	finalMessage := ""
	if !reelsFound {
		finalMessage = fmt.Sprintf("No highlights found for @%s.", userName)
	} else {
		finalMessage = fmt.Sprintf("✅ Finished! Sent %d/%d highlight items for @%s.", atomic.LoadInt64(&processedCount), totalItems, userName)
	}
	c.Telegram.EditMessageText(chatID, sentMsgID, finalMessage)

	return nil
}
