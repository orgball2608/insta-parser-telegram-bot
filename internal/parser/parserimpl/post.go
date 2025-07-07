package paserimpl

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/orgball2608/insta-parser-telegram-bot/pkg/retry"

	"github.com/go-co-op/gocron/v2"
	"github.com/orgball2608/insta-parser-telegram-bot/internal/domain"
	"github.com/orgball2608/insta-parser-telegram-bot/internal/repositories/post"
	"github.com/orgball2608/insta-parser-telegram-bot/pkg/formatter"
)

func (p *ParserImpl) SchedulePostChecking(ctx context.Context) error {
	p.Logger.Info("Setting up post checking scheduler")

	if p.Scheduler == nil {
		loc, err := time.LoadLocation("Asia/Ho_Chi_Minh")
		if err != nil {
			loc = time.Local
			p.Logger.Warn("Failed to load Asia/Ho_Chi_Minh timezone, using local timezone", "error", err)
		}

		scheduler, err := gocron.NewScheduler(gocron.WithLocation(loc))
		if err != nil {
			return fmt.Errorf("failed to create post check scheduler: %w", err)
		}
		p.Scheduler = scheduler
	}

	interval := p.Config.Parser.PostCheckInterval
	p.Logger.Info("Setting up post check interval", "interval", interval)

	_, err := p.Scheduler.NewJob(
		gocron.CronJob(
			interval,
			false,
		),
		gocron.NewTask(func() {
			p.Logger.Info("Running scheduled post check")

			checkCtx, cancel := context.WithTimeout(ctx, 10*time.Minute)
			defer cancel()

			tx, err := p.UnitOfWork.Begin(checkCtx)
			if err != nil {
				p.Logger.Error("Failed to begin transaction for post checking", "error", err)
				return
			}
			defer tx.Rollback(checkCtx)

			usernames, err := tx.Subscription().GetAllUniqueUsernamesByType(checkCtx, domain.SubscriptionTypePost)
			if err != nil {
				p.Logger.Error("Failed to get usernames for post checking", "error", err)
				return
			}

			if len(usernames) == 0 {
				p.Logger.Info("No users subscribed. Skipping.")
				return
			}

			p.Logger.Info("Checking posts for users", "count", len(usernames))

			for _, username := range usernames {
				p.checkNewPostsForUser(checkCtx, username)
			}
		}),
	)

	if err != nil {
		return fmt.Errorf("failed to schedule post checking: %w", err)
	}

	p.Scheduler.Start()

	return nil
}

func (p *ParserImpl) checkNewPostsForUser(ctx context.Context, username string) {
	p.Logger.Info("Checking new posts", "username", username)

	posts, err := p.Instagram.GetUserPosts(ctx, username)
	if err != nil {
		p.Logger.Error("Failed to get posts", "username", username, "error", err)
		return
	}

	p.Logger.Info("Retrieved posts", "username", username, "count", len(posts))

	for _, postItem := range posts {
		tx, err := p.UnitOfWork.Begin(ctx)
		if err != nil {
			p.Logger.Error("Failed to begin transaction for post checking", "error", err)
			continue
		}

		var exists bool
		exists, err = tx.Post().Exists(ctx, postItem.ID)
		if err != nil {
			p.Logger.Error("Failed to check if post exists", "postID", postItem.ID, "error", err)
			tx.Rollback(ctx)
			continue
		}

		if exists {
			p.Logger.Debug("Post already processed", "postID", postItem.ID)
			tx.Rollback(ctx)
			continue
		}

		fullPost, err := p.Instagram.GetUserPost(ctx, postItem.PostURL)
		if err != nil {
			p.Logger.Error("Failed to get post details", "postURL", postItem.PostURL, "error", err)
			tx.Rollback(ctx)
			continue
		}

		postParser := domain.PostParser{
			PostID:   fullPost.ID,
			Username: fullPost.Username,
			PostURL:  fullPost.PostURL,
		}

		err = tx.Post().Create(ctx, postParser)
		if err != nil {
			if !errors.Is(err, post.ErrAlreadyExists) {
				p.Logger.Error("Failed to save post", "postID", fullPost.ID, "error", err)
			}
			tx.Rollback(ctx)
			continue
		}

		subscribers, err := tx.Subscription().GetSubscribersForUserByType(ctx, username, domain.SubscriptionTypePost)
		if err != nil {
			p.Logger.Error("Failed to get subscribers", "username", username, "error", err)
			tx.Rollback(ctx)
			continue
		}

		p.Logger.Info("Sending post to subscribers", "username", username,
			"postID", fullPost.ID, "subscriberCount", len(subscribers))

		for _, chatID := range subscribers {
			p.sendPostToSubscriber(ctx, chatID, fullPost)
		}

		if err := tx.Commit(ctx); err != nil {
			p.Logger.Error("Failed to commit transaction for post checking", "error", err)
		}
	}
}

func (p *ParserImpl) sendPostToSubscriber(ctx context.Context, chatID int64, post *domain.PostItem) {
	escapedUsername := formatter.EscapeMarkdownV2(post.Username)
	escapedCaption := formatter.EscapeMarkdownV2(post.Caption)

	if len(escapedCaption) > 200 {
		escapedCaption = escapedCaption[:197] + "..."
	}

	message := fmt.Sprintf("📢 *New post from @%s*\n\n", escapedUsername)
	if escapedCaption != "" {
		message += fmt.Sprintf("%s\n\n", escapedCaption)
	}

	if strings.Contains(post.PostURL, "/p/") || strings.Contains(post.PostURL, "/reel/") {
		message += fmt.Sprintf("🔗 [View on Instagram](%s)", post.PostURL)
	}

	if len(post.MediaURLs) == 0 {
		sendMessageOperation := func() error {
			_, sendErr := p.Telegram.SendMessage(chatID, message)
			return sendErr
		}
		if err := retry.Do(ctx, p.Logger, "SendMessage", sendMessageOperation, retry.DefaultConfig()); err != nil {
			p.Logger.Error("Failed to send message after retries", "chatID", chatID, "postID", post.ID, "error", err)
		}
		return
	}

	mediaURL := post.MediaURLs[0]
	sendMessageOperation := func() error {
		var sendErr error
		if post.IsVideo {
			_, sendErr = p.Telegram.SendMessage(chatID, fmt.Sprintf("%s\n\n🎬 [Watch Video](%s)", message, mediaURL))
		} else {
			_, sendErr = p.Telegram.SendMessage(chatID, fmt.Sprintf("%s\n\n🖼️ [View Image](%s)", message, mediaURL))
		}
		return sendErr
	}
	if err := retry.Do(ctx, p.Logger, "SendMessage", sendMessageOperation, retry.DefaultConfig()); err != nil {
		p.Logger.Error("Failed to send message after retries", "chatID", chatID, "postID", post.ID, "error", err)
	}
}
