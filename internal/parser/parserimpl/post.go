package paserimpl

import (
	"context"
	"fmt"
	"strings"
	"time"

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

			usernames, err := p.SubscriptionRepo.GetAllUniqueUsernamesByType(checkCtx, domain.SubscriptionTypePost)
			if err != nil {
				p.Logger.Error("Failed to get usernames for post checking", "error", err)
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
		exists, err := p.PostRepo.Exists(ctx, postItem.ID)
		if err != nil {
			p.Logger.Error("Failed to check if post exists", "postID", postItem.ID, "error", err)
			continue
		}

		if exists {
			p.Logger.Debug("Post already processed", "postID", postItem.ID)
			continue
		}

		fullPost, err := p.Instagram.GetUserPost(ctx, postItem.PostURL)
		if err != nil {
			p.Logger.Error("Failed to get post details", "postURL", postItem.PostURL, "error", err)
			continue
		}

		postParser := domain.PostParser{
			PostID:   fullPost.ID,
			Username: fullPost.Username,
			PostURL:  fullPost.PostURL,
		}

		if err := p.PostRepo.Create(ctx, postParser); err != nil {
			if err != post.ErrAlreadyExists {
				p.Logger.Error("Failed to save post", "postID", fullPost.ID, "error", err)
			}
			continue
		}

		subscribers, err := p.SubscriptionRepo.GetSubscribersForUserByType(ctx, username, domain.SubscriptionTypePost)
		if err != nil {
			p.Logger.Error("Failed to get subscribers", "username", username, "error", err)
			continue
		}

		p.Logger.Info("Sending post to subscribers", "username", username, "postID", fullPost.ID, "subscriberCount", len(subscribers))

		for _, chatID := range subscribers {
			p.sendPostToSubscriber(ctx, chatID, fullPost)
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

	if len(post.MediaURLs) > 0 {
		mediaURL := post.MediaURLs[0]

		if post.IsVideo {
			p.Telegram.SendMessage(chatID, fmt.Sprintf("%s\n\n🎬 [Watch Video](%s)", message, mediaURL))
		} else {
			p.Telegram.SendMessage(chatID, fmt.Sprintf("%s\n\n🖼️ [View Image](%s)", message, mediaURL))
		}
	} else {
		p.Telegram.SendMessage(chatID, message)
	}
}
