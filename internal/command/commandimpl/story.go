package commandimpl

import (
	"context"
	"fmt"
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"strings"
	"time"
)

// HandleCommand processes incoming Telegram commands
func (c *CommandImpl) HandleCommand() error {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	u := tgbotapi.NewUpdate(0)
	u.Timeout = 60

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

				// Send error message to user
				_, _ = c.Telegram.SendMessage(update.Message.Chat.ID,
					fmt.Sprintf("Error: %s", err.Error()))
			}
		}
	}

	return nil
}

// processCommand handles individual commands from updates
func (c *CommandImpl) processCommand(ctx context.Context, update tgbotapi.Update) error {
	command := update.Message.Command()

	switch command {
	case "story":
		return c.handleStoryCommand(ctx, update)
	// Add other commands here
	default:
		_, err := c.Telegram.SendMessage(update.Message.Chat.ID,
			"Unknown command. Available commands: /story <username>")
		return err
	}
}

// handleStoryCommand processes the /story command
func (c *CommandImpl) handleStoryCommand(ctx context.Context, update tgbotapi.Update) error {
	// Extract username from command
	args := strings.TrimSpace(strings.TrimPrefix(update.Message.Text, "/story"))
	userName := strings.TrimSpace(args)

	if userName == "" {
		_, err := c.Telegram.SendMessage(update.Message.Chat.ID,
			"Please provide a username: /story <username>")
		return err
	}

	// Send initial response
	_, err := c.Telegram.SendMessage(update.Message.Chat.ID,
		fmt.Sprintf("Getting stories for user: %s...", userName))
	if err != nil {
		return fmt.Errorf("failed to send initial message: %w", err)
	}

	// Set timeout for highlight fetching
	ctxWithTimeout, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	// Get user highlights with context for potential cancellation
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

	// Process each highlight
	for _, highlight := range highlights {
		c.Logger.Info("Processing highlight", "title", highlight.Title, "items", len(highlight.Items))

		if len(highlight.Items) == 0 {
			continue
		}

		// Add cancellation check
		select {
		case <-ctxWithTimeout.Done():
			return fmt.Errorf("operation timed out")
		default:
			if err := c.Parser.ParseStories(highlight.Items); err != nil {
				c.Logger.Error("Error parsing stories", "title", highlight.Title, "error", err)
				continue // Continue with other highlights even if one fails
			}
		}
	}

	// Send a completion message
	_, err = c.Telegram.SendMessage(update.Message.Chat.ID,
		fmt.Sprintf("Processed %d highlights for %s", len(highlights), userName))

	return err
}

func (c *CommandImpl) GetStoryByUserNameCommand() {
	panic("implement")
}
