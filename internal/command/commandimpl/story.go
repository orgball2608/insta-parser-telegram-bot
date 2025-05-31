package commandimpl

import (
	"context"
	"fmt"
	"strings"
	"time"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/orgball2608/insta-parser-telegram-bot/internal/domain"
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
	case "highlights":
		return c.handleHighlightsCommand(ctx, update)
	// Add other commands here
	default:
		_, err := c.Telegram.SendMessage(update.Message.Chat.ID,
			"Unknown command. Available commands:\n"+
				"/story <username> - Get user's current stories\n"+
				"/highlights <username> - Get user highlights")
		return err
	}
}

// handleStoryCommand processes the /story command (formerly currentstory)
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
		fmt.Sprintf("Getting current stories for user: %s...", userName))
	if err != nil {
		return fmt.Errorf("failed to send initial message: %w", err)
	}

	// Set timeout for story fetching
	ctxWithTimeout, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	// Get current user stories
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

	// Clear previous current stories for this user
	if err := c.Parser.ClearCurrentStories(userName); err != nil {
		c.Logger.Error("Error clearing current stories", "error", err)
		// Continue processing even if clearing fails
	}

	processedCount := 0

	// Process each story and save to the CurrentStory repository
	for _, item := range stories {
		// Add cancellation check
		select {
		case <-ctxWithTimeout.Done():
			return fmt.Errorf("operation timed out")
		default:
			// Store media URL
			var mediaURL string
			if len(item.Videos) > 0 {
				mediaURL = item.Videos[0].URL
			} else if len(item.Images.Versions) > 0 {
				mediaURL = item.Images.Versions[0].URL
			}

			if mediaURL == "" {
				continue
			}

			// Create a CurrentStory entity
			currentStory := domain.CurrentStory{
				UserName:  userName,
				MediaURL:  mediaURL,
				CreatedAt: time.Now(),
			}

			// Save to repository
			if err := c.Parser.SaveCurrentStory(currentStory); err != nil {
				c.Logger.Error("Error saving current story", "error", err)
				continue
			}

			// Send media to Telegram
			c.Telegram.SendImageToChanelByUrl(mediaURL)
			processedCount++
		}
	}

	// Send a completion message
	_, err = c.Telegram.SendMessage(update.Message.Chat.ID,
		fmt.Sprintf("Processed %d current stories for %s", processedCount, userName))

	return err
}

// handleHighlightsCommand processes the /highlights command
func (c *CommandImpl) handleHighlightsCommand(ctx context.Context, update tgbotapi.Update) error {
	// Extract username from command
	args := strings.TrimSpace(strings.TrimPrefix(update.Message.Text, "/highlights"))
	userName := strings.TrimSpace(args)

	if userName == "" {
		_, err := c.Telegram.SendMessage(update.Message.Chat.ID,
			"Please provide a username: /highlights <username>")
		return err
	}

	// Send initial response
	_, err := c.Telegram.SendMessage(update.Message.Chat.ID,
		fmt.Sprintf("Getting highlights for user: %s...", userName))
	if err != nil {
		return fmt.Errorf("failed to send initial message: %w", err)
	}

	// Set timeout for highlight fetching
	ctxWithTimeout, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	// Get user highlights
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

	// Process each highlight and save to the Highlights repository
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
			// Process each item in the highlight
			for _, item := range highlight.Items {
				// Store media URL
				var mediaURL string
				if len(item.Videos) > 0 {
					mediaURL = item.Videos[0].URL
				} else if len(item.Images.Versions) > 0 {
					mediaURL = item.Images.Versions[0].URL
				}

				if mediaURL == "" {
					continue
				}

				// Create a Highlights entity
				highlightItem := domain.Highlights{
					UserName:  userName,
					MediaURL:  mediaURL,
					CreatedAt: time.Now(),
				}

				// Save to repository
				if err := c.Parser.SaveHighlight(highlightItem); err != nil {
					c.Logger.Error("Error saving highlight", "error", err)
					continue
				}

				// Send media to Telegram
				c.Telegram.SendImageToChanelByUrl(mediaURL)
				processedCount++
			}
		}
	}

	// Send a completion message
	_, err = c.Telegram.SendMessage(update.Message.Chat.ID,
		fmt.Sprintf("Processed %d highlight items for %s", processedCount, userName))

	return err
}
