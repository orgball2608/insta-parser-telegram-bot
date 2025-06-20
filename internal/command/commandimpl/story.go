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

						_, _ = c.Telegram.SendMessage(u.Message.Chat.ID,
							fmt.Sprintf("An error occurred: %s", err.Error()))
					}
				}
			}(update)
		}
	}
}

func (c *CommandImpl) processCommand(ctx context.Context, update tgbotapi.Update) error {
	command := update.Message.Command()

	switch command {
	case "story":
		return c.handleStoryCommand(ctx, update)
	case "highlights":
		return c.handleHighlightsCommand(ctx, update)
	case "post":
		return c.handlePostCommand(ctx, update)
	default:
		_, err := c.Telegram.SendMessage(update.Message.Chat.ID,
			"Unknown command. Available commands:\n"+
				"/story <username> - Get user's current stories\n"+
				"/highlights <username> - Get user highlights\n"+
				"/post <post_url> - Get post from URL")
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

	_, err := c.Telegram.SendMessage(chatID, fmt.Sprintf("Getting current stories for user: %s...", userName))
	if err != nil {
		return fmt.Errorf("failed to send initial message: %w", err)
	}

	stories, err := c.Instagram.GetUserStories(userName)
	if err != nil {
		if errors.Is(err, instagram.ErrPrivateAccount) {
			_, _ = c.Telegram.SendMessage(chatID, fmt.Sprintf("Account '%s' is private. I cannot fetch stories.", userName))
			return nil
		}
		return fmt.Errorf("failed to get stories for %s: %w", userName, err)
	}

	if len(stories) == 0 {
		_, err := c.Telegram.SendMessage(chatID, fmt.Sprintf("No current stories found for user: %s", userName))
		return err
	}

	if err := c.Parser.ClearCurrentStories(userName); err != nil {
		c.Logger.Error("Error clearing current stories", "error", err)
	}

	var processedCount int
	for _, item := range stories {
		select {
		case <-ctx.Done():
			return fmt.Errorf("operation timed out")
		default:
			if item.MediaURL == "" {
				continue
			}

			currentStory := domain.CurrentStory{
				UserName:  userName,
				MediaURL:  item.MediaURL,
				CreatedAt: time.Now(),
			}

			if err := c.Parser.SaveCurrentStory(currentStory); err != nil {
				c.Logger.Error("Error saving current story", "error", err)
				continue
			}

			if err := c.Telegram.SendMediaByUrl(chatID, item.MediaURL); err != nil {
				c.Logger.Error("Failed to send story media", "url", item.MediaURL, "error", err)
			} else {
				processedCount++
			}
		}
	}

	_, err = c.Telegram.SendMessage(chatID, fmt.Sprintf("Processed %d current stories for %s", processedCount, userName))
	return err
}

func (c *CommandImpl) handleHighlightsCommand(ctx context.Context, update tgbotapi.Update) error {
	args := strings.TrimSpace(update.Message.CommandArguments())
	userName := strings.TrimSpace(args)
	chatID := update.Message.Chat.ID

	if userName == "" {
		_, err := c.Telegram.SendMessage(chatID, "Please provide a username: /highlights <username>")
		return err
	}

	_, err := c.Telegram.SendMessage(chatID, fmt.Sprintf("Getting highlights for @%s... This may take a while.", userName))
	if err != nil {
		return fmt.Errorf("failed to send initial message: %w", err)
	}

	var processedCount int64
	var reelsFound bool

	processor := func(highlightReel domain.HighlightReel) error {
		reelsFound = true
		c.Logger.Info("Processing highlight reel", "title", highlightReel.Title, "items", len(highlightReel.Items))

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

	err = c.Instagram.GetUserHighlights(userName, processor)
	if err != nil {
		if errors.Is(err, instagram.ErrPrivateAccount) {
			_, _ = c.Telegram.SendMessage(chatID, fmt.Sprintf("Account '@%s' is private. I cannot fetch highlights.", userName))
			return nil
		}
		return fmt.Errorf("failed to get highlights for @%s: %w", userName, err)
	}

	if !reelsFound {
		_, err = c.Telegram.SendMessage(chatID, fmt.Sprintf("No highlights found for user: @%s", userName))
	} else {
		_, err = c.Telegram.SendMessage(chatID, fmt.Sprintf("Finished processing. Sent %d highlight items for @%s.", atomic.LoadInt64(&processedCount), userName))
	}

	return err
}
