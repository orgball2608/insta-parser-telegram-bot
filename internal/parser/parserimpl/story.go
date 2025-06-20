package paserimpl

import (
	"context"
	"errors"
	"fmt"
	"math/rand"
	"sync"
	"time"

	"github.com/go-co-op/gocron/v2"
	"github.com/orgball2608/insta-parser-telegram-bot/internal/domain"
	storyRepo "github.com/orgball2608/insta-parser-telegram-bot/internal/repositories/story"
	"github.com/panjf2000/ants/v2"
)

func (p *ParserImpl) ScheduleParseStories(ctx context.Context) error {
	loc, err := time.LoadLocation("Asia/Ho_Chi_Minh")
	if err != nil {
		loc = time.Local
		p.Logger.Warn("Failed to load Asia/Ho_Chi_Minh timezone, using local timezone", "error", err)
	}

	scheduler, err := gocron.NewScheduler(gocron.WithLocation(loc))
	if err != nil {
		return fmt.Errorf("failed to create scheduler: %w", err)
	}

	_, err = scheduler.NewJob(
		gocron.DurationRandomJob(15*time.Minute, 20*time.Minute),
		gocron.NewTask(func() {
			if ctx.Err() != nil {
				p.Logger.Info("Context cancelled, stopping story parsing schedule")
				return
			}
			taskCtx, cancel := context.WithTimeout(ctx, 10*time.Minute)
			defer cancel()

			p.Logger.Info("Starting scheduled story parsing for subscribed users...")

			usernames, err := p.SubscriptionRepo.GetAllUniqueUsernames(taskCtx)
			if err != nil {
				p.Logger.Error("Failed to get unique usernames from subscriptions", "error", err)
				return
			}

			if len(usernames) == 0 {
				p.Logger.Info("No users subscribed. Skipping.")
				return
			}

			p.Logger.Info("Found users to parse", "count", len(usernames))
			shuffledUsernames := shuffleUsernames(usernames)

			p.runJobsWithAnts(taskCtx, shuffledUsernames)

			p.Logger.Info("Completed scheduling all jobs for this run.")
		}),
	)
	if err != nil {
		return fmt.Errorf("failed to schedule story parsing: %w", err)
	}

	scheduler.Start()

	go func() {
		<-ctx.Done()
		p.Logger.Info("Stopping story parsing scheduler")
		if err := scheduler.Shutdown(); err != nil {
			p.Logger.Error("Failed to shut down scheduler", "error", err)
		}
	}()

	return nil
}

func (p *ParserImpl) runJobsWithAnts(ctx context.Context, usernames []string) {
	var wg sync.WaitGroup
	pool, _ := ants.NewPool(5, ants.WithPreAlloc(true))
	defer pool.Release()

	for _, username := range usernames {
		wg.Add(1)
		userToProcess := username

		err := pool.Submit(func() {
			defer wg.Done()
			select {
			case <-ctx.Done():
				p.Logger.Info("Skipping job due to context cancellation", "username", userToProcess)
				return
			default:
				p.Logger.Info("Worker processing user", "username", userToProcess)
				if err := p.processSubscribedUser(ctx, userToProcess); err != nil {
					p.Logger.Error("Worker failed to process user", "username", userToProcess, "error", err)
				} else {
					p.Logger.Info("Worker successfully processed user", "username", userToProcess)
				}
				time.Sleep(time.Duration(1+rand.Intn(3)) * time.Second)
			}
		})
		if err != nil {
			wg.Done()
			p.Logger.Error("Failed to submit job to ants pool", "username", userToProcess, "error", err)
		}
	}

	wg.Wait()
}

func (p *ParserImpl) processSubscribedUser(ctx context.Context, username string) error {
	stories, err := p.Instagram.GetUserStories(username)
	if err != nil {
		return fmt.Errorf("failed to get stories for %s: %w", username, err)
	}

	if len(stories) == 0 {
		p.Logger.Info("No stories found for user", "username", username)
		return nil
	}

	var newStories []domain.StoryItem
	for _, story := range stories {
		exists, err := p.checkStoryExists(story.ID)
		if err != nil {
			p.Logger.Error("Failed to check story existence", "story_id", story.ID, "error", err)
			continue
		}
		if !exists {
			newStories = append(newStories, story)
		}
	}

	if len(newStories) == 0 {
		p.Logger.Info("No new stories for user", "username", username)
		return nil
	}

	p.Logger.Info("Found new stories", "username", username, "count", len(newStories))

	subscriberIDs, err := p.SubscriptionRepo.GetSubscribersForUser(ctx, username)
	if err != nil {
		return fmt.Errorf("failed to get subscribers for %s: %w", username, err)
	}

	if len(subscriberIDs) == 0 {
		p.Logger.Warn("Found new stories but no one is subscribed", "username", username)
		return nil
	}

	for _, story := range newStories {
		dbStory := domain.Story{
			StoryID:   story.ID,
			UserName:  story.Username,
			CreatedAt: story.TakenAt,
		}
		if err := p.StoryRepo.Create(ctx, dbStory); err != nil {
			if errors.Is(err, storyRepo.ErrCannotCreate) {
				p.Logger.Warn("Story might already exist or failed to create, skipping send", "story_id", dbStory.StoryID)
				continue
			}
			p.Logger.Error("Failed to save story to DB", "story_id", dbStory.StoryID, "error", err)
			continue
		}

		for _, chatID := range subscriberIDs {
			err := p.Telegram.SendMediaByUrl(chatID, story.MediaURL)
			if err != nil {
				p.Logger.Error("Failed to send story to subscriber", "chat_id", chatID, "url", story.MediaURL, "error", err)
			}
		}
		time.Sleep(time.Duration(1500+rand.Intn(2000)) * time.Millisecond)
	}

	return nil
}

func shuffleUsernames(usernames []string) []string {
	result := make([]string, len(usernames))
	copy(result, usernames)
	r := rand.New(rand.NewSource(time.Now().UnixNano()))
	r.Shuffle(len(result), func(i, j int) {
		result[i], result[j] = result[j], result[i]
	})
	return result
}

func (p *ParserImpl) ParseUserStories(_ context.Context, username string) error {
	p.Logger.Info("Parsing user stories", "username", username)

	stories, err := p.Instagram.GetUserStories(username)
	if err != nil {
		return fmt.Errorf("failed to get stories for %s: %w", username, err)
	}

	if len(stories) == 0 {
		p.Logger.Info("No stories found for user", "username", username)
		return nil
	}

	p.Logger.Info("Found stories", "username", username, "count", len(stories))
	return p.ProcessStories(stories)
}

func (p *ParserImpl) ProcessStories(stories []domain.StoryItem) error {
	if len(stories) == 0 {
		return nil
	}

	semaphore := make(chan struct{}, 3)
	var wg sync.WaitGroup
	var errsMutex sync.Mutex
	var errs []error
	var processed, skipped, failed int
	var statsMutex sync.Mutex

	p.Logger.Info("Processing stories", "count", len(stories))

	for _, story := range stories {
		wg.Add(1)
		semaphore <- struct{}{}

		go func(item domain.StoryItem) {
			defer func() {
				<-semaphore
				wg.Done()
				if r := recover(); r != nil {
					errsMutex.Lock()
					errs = append(errs, fmt.Errorf("panic in story processing: %v", r))
					errsMutex.Unlock()
					statsMutex.Lock()
					failed++
					statsMutex.Unlock()
				}
			}()

			exists, err := p.checkStoryExists(item.ID)
			if err != nil {
				errsMutex.Lock()
				errs = append(errs, fmt.Errorf("failed to check story existence: %w", err))
				errsMutex.Unlock()
				statsMutex.Lock()
				failed++
				statsMutex.Unlock()
				return
			}

			if exists {
				p.Logger.Debug("Story already processed", "storyID", item.ID)
				statsMutex.Lock()
				skipped++
				statsMutex.Unlock()
				return
			}

			if err := p.processStoryItem(item); err != nil {
				errsMutex.Lock()
				errs = append(errs, fmt.Errorf("failed to process story %s: %w", item.ID, err))
				errsMutex.Unlock()
				statsMutex.Lock()
				failed++
				statsMutex.Unlock()
				return
			}

			statsMutex.Lock()
			processed++
			statsMutex.Unlock()
		}(story)
	}

	wg.Wait()
	close(semaphore)

	p.Logger.Info("Story processing completed", "total", len(stories), "processed", processed, "skipped", skipped, "failed", failed)

	if len(errs) > 0 {
		return fmt.Errorf("encountered %d errors during story parsing, first error: %w", len(errs), errs[0])
	}
	return nil
}

func (p *ParserImpl) checkStoryExists(storyID string) (bool, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if storyID == "" {
		p.Logger.Warn("checkStoryExists called with empty storyID")
		return true, nil
	}

	_, err := p.StoryRepo.GetByStoryID(ctx, storyID)
	if err != nil {
		if errors.Is(err, storyRepo.ErrNotFound) {
			return false, nil
		}
		return false, err
	}
	return true, nil
}

func (p *ParserImpl) processStoryItem(item domain.StoryItem) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	story := domain.Story{
		StoryID:   item.ID,
		UserName:  item.Username,
		CreatedAt: item.TakenAt,
	}

	if err := p.StoryRepo.Create(ctx, story); err != nil {
		if errors.Is(err, storyRepo.ErrCannotCreate) {
			p.Logger.Warn("Story might already exist or failed to create, skipping send", "story_id", story.StoryID)
			return nil
		}
		return fmt.Errorf("failed to save story: %w", err)
	}

	p.Logger.Info("Processing media item", "username", item.Username, "url", item.MediaURL, "type", item.MediaType)
	p.Telegram.SendMediaToDefaultChannelByUrl(item.MediaURL)

	delay := time.Duration(1500+rand.Intn(2000)) * time.Millisecond
	p.Logger.Info("Scheduled job: Waiting to avoid rate limit", "delay", delay)
	time.Sleep(delay)
	return nil
}

func (p *ParserImpl) SaveHighlight(highlight domain.Highlights) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	p.Logger.Info("Saving highlight", "username", highlight.UserName, "mediaURL", highlight.MediaURL)
	err := p.HighlightsRepo.Create(ctx, highlight)
	if err != nil {
		return fmt.Errorf("failed to save highlight: %w", err)
	}
	return nil
}

func (p *ParserImpl) SaveCurrentStory(currentStory domain.CurrentStory) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	p.Logger.Info("Saving current story", "username", currentStory.UserName, "mediaURL", currentStory.MediaURL)
	err := p.CurrentStoryRepo.Create(ctx, currentStory)
	if err != nil {
		return fmt.Errorf("failed to save current story: %w", err)
	}
	return nil
}

func (p *ParserImpl) ClearCurrentStories(username string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	p.Logger.Info("Clearing current stories for user", "username", username)
	err := p.CurrentStoryRepo.DeleteByUserName(ctx, username)
	if err != nil {
		return fmt.Errorf("failed to clear current stories for %s: %w", username, err)
	}
	return nil
}
