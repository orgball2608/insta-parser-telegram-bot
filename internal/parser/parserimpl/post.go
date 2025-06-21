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

// SchedulePostChecking sets up a scheduler to check for new posts
func (p *ParserImpl) SchedulePostChecking(ctx context.Context) error {
	p.Logger.Info("Setting up post checking scheduler")

	// Create a scheduler if not already created
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

	// Parse the interval from config
	interval := p.Config.Parser.PostCheckInterval
	p.Logger.Info("Setting up post check interval", "interval", interval)

	// Create a job with the specified interval
	_, err := p.Scheduler.NewJob(
		gocron.CronJob(
			interval,
			false, // Don't use seconds precision
		),
		gocron.NewTask(func() {
			p.Logger.Info("Running scheduled post check")

			// Create a context with timeout for the post checking operation
			checkCtx, cancel := context.WithTimeout(ctx, 10*time.Minute)
			defer cancel()

			// Get all usernames with post or all subscription types
			usernames, err := p.SubscriptionRepo.GetAllUniqueUsernamesByType(checkCtx, domain.SubscriptionTypePost)
			if err != nil {
				p.Logger.Error("Failed to get usernames for post checking", "error", err)
				return
			}

			p.Logger.Info("Checking posts for users", "count", len(usernames))

			// Process each username
			for _, username := range usernames {
				p.checkNewPostsForUser(checkCtx, username)
			}
		}),
	)

	if err != nil {
		return fmt.Errorf("failed to schedule post checking: %w", err)
	}

	// Start the scheduler if not already started
	p.Scheduler.Start()

	return nil
}

// checkNewPostsForUser checks for new posts for a specific user
func (p *ParserImpl) checkNewPostsForUser(ctx context.Context, username string) {
	p.Logger.Info("Checking new posts", "username", username)

	// Get the latest posts from Instagram
	posts, err := p.Instagram.GetUserPosts(ctx, username)
	if err != nil {
		p.Logger.Error("Failed to get posts", "username", username, "error", err)
		return
	}

	p.Logger.Info("Retrieved posts", "username", username, "count", len(posts))

	// Process each post
	for _, postItem := range posts {
		// Check if we've already processed this post
		exists, err := p.PostRepo.Exists(ctx, postItem.ID)
		if err != nil {
			p.Logger.Error("Failed to check if post exists", "postID", postItem.ID, "error", err)
			continue
		}

		if exists {
			p.Logger.Debug("Post already processed", "postID", postItem.ID)
			continue
		}

		// This is a new post, get full details
		fullPost, err := p.Instagram.GetUserPost(ctx, postItem.PostURL)
		if err != nil {
			p.Logger.Error("Failed to get post details", "postURL", postItem.PostURL, "error", err)
			continue
		}

		// Save the post to the database
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

		// Get subscribers for this username who want post updates
		subscribers, err := p.SubscriptionRepo.GetSubscribersForUserByType(ctx, username, domain.SubscriptionTypePost)
		if err != nil {
			p.Logger.Error("Failed to get subscribers", "username", username, "error", err)
			continue
		}

		p.Logger.Info("Sending post to subscribers", "username", username, "postID", fullPost.ID, "subscriberCount", len(subscribers))

		// Send the post to each subscriber
		for _, chatID := range subscribers {
			p.sendPostToSubscriber(ctx, chatID, fullPost)
		}
	}
}

// sendPostToSubscriber sends a post to a subscriber
func (p *ParserImpl) sendPostToSubscriber(ctx context.Context, chatID int64, post *domain.PostItem) {
	// Escape username and caption for Markdown
	escapedUsername := formatter.EscapeMarkdownV2(post.Username)
	escapedCaption := formatter.EscapeMarkdownV2(post.Caption)

	// Truncate caption if too long
	if len(escapedCaption) > 200 {
		escapedCaption = escapedCaption[:197] + "..."
	}

	// Create message text
	message := fmt.Sprintf("📢 *New post from @%s*\n\n", escapedUsername)
	if escapedCaption != "" {
		message += fmt.Sprintf("%s\n\n", escapedCaption)
	}

	// Only add the "View on Instagram" link if the URL contains "/p/" or "/reel/"
	if strings.Contains(post.PostURL, "/p/") || strings.Contains(post.PostURL, "/reel/") {
		message += fmt.Sprintf("🔗 [View on Instagram](%s)", post.PostURL)
	}

	// Send message with media if available
	if len(post.MediaURLs) > 0 {
		// For simplicity, just send the first media item
		// In a real implementation, you might want to send all media items as an album
		mediaURL := post.MediaURLs[0]

		if post.IsVideo {
			// Use SendMessage with the URL if SendVideo is not available
			p.Telegram.SendMessage(chatID, fmt.Sprintf("%s\n\n🎬 [Watch Video](%s)", message, mediaURL))
		} else {
			// Use SendMessage with the URL if SendPhoto is not available
			p.Telegram.SendMessage(chatID, fmt.Sprintf("%s\n\n🖼️ [View Image](%s)", message, mediaURL))
		}
	} else {
		// No media, just send the message
		p.Telegram.SendMessage(chatID, message)
	}
}
