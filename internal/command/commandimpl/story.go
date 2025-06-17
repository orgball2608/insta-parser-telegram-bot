package commandimpl

import (
	"context"
	"fmt"
	"strings"
	"time"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/orgball2608/insta-parser-telegram-bot/internal/domain"
)

func (c *CommandImpl) HandleCommand() error {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	u := tgbotapi.NewUpdate(0)

	updates, err := c.Telegram.GetUpdatesChan(u)
	if err != nil {
		c.Logger.Error("Error getting updates from telegram", "error", err)
		return fmt.Errorf("failed to get telegram updates: %w", err)
	}

	c.Logger.Info("Starting command handler, waiting for messages")

	for update := range updates {
		if update.Message == nil {
			continue
		}

		c.Logger.Info("Message received",
			"from", update.Message.From.UserName,
			"text", update.Message.Text)

		if update.Message.IsCommand() {
			if err := c.processCommand(ctx, update); err != nil {
				c.Logger.Error("Error processing command",
					"command", update.Message.Command(),
					"error", err)

				_, _ = c.Telegram.SendMessage(update.Message.Chat.ID,
					fmt.Sprintf("Error: %s", err.Error()))
			}
		}
	}

	return nil
}

func (c *CommandImpl) processCommand(ctx context.Context, update tgbotapi.Update) error {
	command := update.Message.Command()

	switch command {
	case "story":
		return c.handleStoryCommand(ctx, update)
	case "highlights":
		return c.handleHighlightsCommand(ctx, update)
	default:
		_, err := c.Telegram.SendMessage(update.Message.Chat.ID,
			"Unknown command. Available commands:\n"+
				"/story <username> - Get user's current stories\n"+
				"/highlights <username> - Get user highlights")
		return err
	}
}

func (c *CommandImpl) handleStoryCommand(ctx context.Context, update tgbotapi.Update) error {
	args := strings.TrimSpace(strings.TrimPrefix(update.Message.Text, "/story"))
	userName := strings.TrimSpace(args)

	if userName == "" {
		_, err := c.Telegram.SendMessage(update.Message.Chat.ID,
			"Please provide a username: /story <username>")
		return err
	}

	_, err := c.Telegram.SendMessage(update.Message.Chat.ID,
		fmt.Sprintf("Getting current stories for user: %s...", userName))
	if err != nil {
		return fmt.Errorf("failed to send initial message: %w", err)
	}

	ctxWithTimeout, cancel := context.WithTimeout(ctx, 15*time.Minute)
	defer cancel()

	stories, err := c.Instagram.GetUserStories(userName)
	if err != nil {
		return fmt.Errorf("failed to get stories for %s: %w", userName, err)
	}

	c.Logger.Info("Retrieved stories", "username", userName, "count", len(stories))

	if len(stories) == 0 {
		_, err := c.Telegram.SendMessage(update.Message.Chat.ID,
			fmt.Sprintf("No current stories found for user: %s", userName))
		return err
	}

	if err := c.Parser.ClearCurrentStories(userName); err != nil {
		c.Logger.Error("Error clearing current stories", "error", err)
	}

	processedCount := 0

	for _, item := range stories {
		select {
		case <-ctxWithTimeout.Done():
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

			c.Telegram.SendMediaToChanelByUrl(item.MediaURL)
			processedCount++
		}
	}

	_, err = c.Telegram.SendMessage(update.Message.Chat.ID,
		fmt.Sprintf("Processed %d current stories for %s", processedCount, userName))

	return err
}

func (c *CommandImpl) handleHighlightsCommand(ctx context.Context, update tgbotapi.Update) error {
	args := strings.TrimSpace(strings.TrimPrefix(update.Message.Text, "/highlights"))
	userName := strings.TrimSpace(args)

	if userName == "" {
		_, err := c.Telegram.SendMessage(update.Message.Chat.ID,
			"Please provide a username: /highlights <username>")
		return err
	}

	_, err := c.Telegram.SendMessage(update.Message.Chat.ID,
		fmt.Sprintf("Getting highlights for user: %s...", userName))
	if err != nil {
		return fmt.Errorf("failed to send initial message: %w", err)
	}

	ctxWithTimeout, cancel := context.WithTimeout(ctx, 15*time.Minute)
	defer cancel()

	highlights, err := c.Instagram.GetUserHighlights(userName)
	if err != nil {
		return fmt.Errorf("failed to get highlights for %s: %w", userName, err)
	}

	c.Logger.Info("Retrieved highlights", "username", userName, "count", len(highlights))

	if len(highlights) == 0 {
		_, err := c.Telegram.SendMessage(update.Message.Chat.ID,
			fmt.Sprintf("No highlights found for user: %s", userName))
		return err
	}

	processedCount := 0

	for _, highlightReel := range highlights {
		c.Logger.Info("Processing highlight", "title", highlightReel.Title, "items", len(highlightReel.Items))

		if len(highlightReel.Items) == 0 {
			continue
		}

		select {
		case <-ctxWithTimeout.Done():
			return fmt.Errorf("operation timed out")
		default:
			for _, item := range highlightReel.Items {
				if item.MediaURL == "" {
					continue
				}

				highlightItem := domain.Highlights{
					UserName:  userName,
					MediaURL:  item.MediaURL,
					CreatedAt: time.Now(),
				}

				if err := c.Parser.SaveHighlight(highlightItem); err != nil {
					c.Logger.Error("Error saving highlight", "error", err)
					continue
				}

				c.Telegram.SendMediaToChanelByUrl(item.MediaURL)
				processedCount++
			}
		}
	}

	_, err = c.Telegram.SendMessage(update.Message.Chat.ID,
		fmt.Sprintf("Processed %d highlight items for %s", processedCount, userName))

	return err
}
