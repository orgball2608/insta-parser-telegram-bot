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
		c.Telegram.SendMessage(chatID, "Vui lòng nhập username. Ví dụ: /subscribe <username>")
		return
	}

	sub := domain.Subscription{
		ChatID:            chatID,
		InstagramUsername: username,
	}

	err := c.SubscriptionRepo.Create(ctx, sub)
	if err != nil {
		if errors.Is(err, subscription.ErrAlreadyExists) {
			c.Telegram.SendMessage(chatID, fmt.Sprintf("Bạn đã đăng ký theo dõi @%s từ trước.", username))
		} else {
			c.Logger.Error("Failed to create subscription", "error", err)
			c.Telegram.SendMessage(chatID, "Đã có lỗi xảy ra. Vui lòng thử lại sau.")
		}
		return
	}

	c.Telegram.SendMessage(chatID, fmt.Sprintf("✅ Đăng ký thành công! Bạn sẽ nhận được story mới từ @%s.", username))
}

func (c *CommandImpl) handleUnsubscribe(ctx context.Context, chatID int64, args string) {
	username := subscription.SanitizeUsername(args)
	if username == "" {
		c.Telegram.SendMessage(chatID, "Vui lòng nhập username. Ví dụ: /unsubscribe <username>")
		return
	}

	err := c.SubscriptionRepo.Delete(ctx, chatID, username)
	if err != nil {
		if errors.Is(err, subscription.ErrNotFound) {
			c.Telegram.SendMessage(chatID, fmt.Sprintf("Bạn chưa đăng ký theo dõi @%s.", username))
		} else {
			c.Logger.Error("Failed to delete subscription", "error", err)
			c.Telegram.SendMessage(chatID, "Đã có lỗi xảy ra. Vui lòng thử lại sau.")
		}
		return
	}

	c.Telegram.SendMessage(chatID, fmt.Sprintf("Đã hủy theo dõi @%s.", username))
}

func (c *CommandImpl) handleListSubscriptions(ctx context.Context, chatID int64) {
	subs, err := c.SubscriptionRepo.GetByChatID(ctx, chatID)
	if err != nil {
		c.Logger.Error("Failed to get subscriptions", "error", err)
		c.Telegram.SendMessage(chatID, "Đã có lỗi xảy ra khi lấy danh sách.")
		return
	}

	if len(subs) == 0 {
		c.Telegram.SendMessage(chatID, "Bạn chưa đăng ký theo dõi tài khoản nào. Dùng /subscribe để bắt đầu.")
		return
	}

	var builder strings.Builder
	builder.WriteString("📝 **Danh sách bạn đang theo dõi:**\n")
	for i, sub := range subs {
		builder.WriteString(fmt.Sprintf("%d. @%s\n", i+1, sub.InstagramUsername))
	}

	c.Telegram.SendMessage(chatID, builder.String())
}
