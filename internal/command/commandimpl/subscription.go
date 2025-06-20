package commandimpl

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/orgball2608/insta-parser-telegram-bot/internal/domain"
	"github.com/orgball2608/insta-parser-telegram-bot/internal/repositories/subscription"
)

func (c *CommandImpl) handleSubscribe(ctx context.Context, chatID int64, args string) {
	username := subscription.SanitizeUsername(args)
	if username == "" {
		c.Telegram.SendMessage(chatID, "Please provide a username. Usage: /subscribe <username>")
		return
	}

	sub := domain.Subscription{
		ChatID:            chatID,
		InstagramUsername: username,
	}

	err := c.SubscriptionRepo.Create(ctx, sub)
	if err != nil {
		if errors.Is(err, subscription.ErrAlreadyExists) {
			c.Telegram.SendMessage(chatID, fmt.Sprintf("You are already subscribed to @%s.", username))
		} else {
			c.Logger.Error("Failed to create subscription", "error", err)
			c.Telegram.SendMessage(chatID, "An error occurred. Please try again later.")
		}
		return
	}

	c.Telegram.SendMessage(chatID, fmt.Sprintf("✅ Successfully subscribed! You will now receive new stories from @%s.", username))
}

func (c *CommandImpl) handleUnsubscribe(ctx context.Context, chatID int64, args string) {
	username := subscription.SanitizeUsername(args)
	if username == "" {
		c.Telegram.SendMessage(chatID, "Please provide a username. Usage: /unsubscribe <username>")
		return
	}

	err := c.SubscriptionRepo.Delete(ctx, chatID, username)
	if err != nil {
		if errors.Is(err, subscription.ErrNotFound) {
			c.Telegram.SendMessage(chatID, fmt.Sprintf("You are not subscribed to @%s.", username))
		} else {
			c.Logger.Error("Failed to delete subscription", "error", err)
			c.Telegram.SendMessage(chatID, "An error occurred. Please try again later.")
		}
		return
	}

	c.Telegram.SendMessage(chatID, fmt.Sprintf("Successfully unsubscribed from @%s.", username))
}

func (c *CommandImpl) handleListSubscriptions(ctx context.Context, chatID int64) {
	subs, err := c.SubscriptionRepo.GetByChatID(ctx, chatID)
	if err != nil {
		c.Logger.Error("Failed to get subscriptions", "error", err)
		c.Telegram.SendMessage(chatID, "An error occurred while fetching your subscriptions.")
		return
	}

	if len(subs) == 0 {
		c.Telegram.SendMessage(chatID, "You are not subscribed to any accounts. Use /subscribe to start.")
		return
	}

	var builder strings.Builder
	builder.WriteString("📝 *You are currently subscribed to:* \n")
	for i, sub := range subs {
		builder.WriteString(fmt.Sprintf("%d. @%s\n", i+1, sub.InstagramUsername))
	}

	c.Telegram.SendMessage(chatID, builder.String())
}
