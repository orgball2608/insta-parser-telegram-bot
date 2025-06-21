package commandimpl

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/orgball2608/insta-parser-telegram-bot/internal/domain"
	"github.com/orgball2608/insta-parser-telegram-bot/internal/repositories/subscription"
	"github.com/orgball2608/insta-parser-telegram-bot/pkg/formatter"
)

func (c *CommandImpl) handleSubscribe(ctx context.Context, chatID int64, args string) {
	parts := strings.Fields(args)
	if len(parts) == 0 {
		c.Telegram.SendMessage(chatID, "Please provide a username. Usage: /subscribe <username> [post|story|all]")
		return
	}

	username := subscription.SanitizeUsername(parts[0])
	if username == "" {
		c.Telegram.SendMessage(chatID, "Please provide a valid username. Usage: /subscribe <username> [post|story|all]")
		return
	}

	subscriptionType := domain.SubscriptionTypeStory

	if len(parts) > 1 {
		specifiedType := strings.ToLower(parts[1])
		if domain.IsValidSubscriptionType(specifiedType) {
			subscriptionType = specifiedType
		} else {
			c.Telegram.SendMessage(chatID, "Invalid subscription type. Valid types are: post, story, all. Using default: story.")
		}
	}

	escapedUsername := formatter.EscapeMarkdownV2(username)
	sub := domain.Subscription{
		ChatID:            chatID,
		InstagramUsername: username,
		SubscriptionType:  subscriptionType,
	}

	err := c.SubscriptionRepo.Create(ctx, sub)
	if err != nil {
		if errors.Is(err, subscription.ErrAlreadyExists) {
			err = c.SubscriptionRepo.UpdateSubscriptionType(ctx, chatID, username, subscriptionType)
			if err != nil {
				c.Logger.Error("Failed to update subscription type", "error", err)
				c.Telegram.SendMessage(chatID, "You are already subscribed to this account. Failed to update subscription type.")
				return
			}
			c.Telegram.SendMessage(chatID, fmt.Sprintf("Updated subscription type to '%s' for @%s.", subscriptionType, escapedUsername))
		} else {
			c.Logger.Error("Failed to create subscription", "error", err)
			c.Telegram.SendMessage(chatID, "An error occurred. Please try again later.")
		}
		return
	}

	var contentType string
	switch subscriptionType {
	case domain.SubscriptionTypePost:
		contentType = "posts"
	case domain.SubscriptionTypeStory:
		contentType = "stories"
	case domain.SubscriptionTypeAll:
		contentType = "posts and stories"
	}

	c.Telegram.SendMessage(chatID, fmt.Sprintf("✅ Successfully subscribed! You will now receive new %s from @%s.", contentType, escapedUsername))
}

func (c *CommandImpl) handleUnsubscribe(ctx context.Context, chatID int64, args string) {
	username := subscription.SanitizeUsername(args)
	if username == "" {
		c.Telegram.SendMessage(chatID, "Please provide a username. Usage: /unsubscribe <username>")
		return
	}

	escapedUsername := formatter.EscapeMarkdownV2(username)
	err := c.SubscriptionRepo.Delete(ctx, chatID, username)
	if err != nil {
		if errors.Is(err, subscription.ErrNotFound) {
			c.Telegram.SendMessage(chatID, fmt.Sprintf("You are not subscribed to @%s.", escapedUsername))
		} else {
			c.Logger.Error("Failed to delete subscription", "error", err)
			c.Telegram.SendMessage(chatID, "An error occurred. Please try again later.")
		}
		return
	}

	c.Telegram.SendMessage(chatID, fmt.Sprintf("Successfully unsubscribed from @%s.", escapedUsername))
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
		escapedUsername := formatter.EscapeMarkdownV2(sub.InstagramUsername)

		subscriptionInfo := fmt.Sprintf("%d. @%s (%s)", i+1, escapedUsername, sub.SubscriptionType)
		builder.WriteString(subscriptionInfo + "\n")
	}

	builder.WriteString("\n*Available subscription types:*\n")
	builder.WriteString("• story - receive only stories\n")
	builder.WriteString("• post - receive only posts\n")
	builder.WriteString("• all - receive both posts and stories\n\n")
	builder.WriteString("To change subscription type: /subscribe <username> <type>")

	c.Telegram.SendMessage(chatID, builder.String())
}
