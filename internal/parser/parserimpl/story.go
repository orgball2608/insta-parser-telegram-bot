package paserimpl

import (
	"context"
	"errors"
	"fmt"
	"math/rand"
	"sync"
	"time"

	"github.com/orgball2608/insta-parser-telegram-bot/internal/unitofwork"
	"github.com/orgball2608/insta-parser-telegram-bot/pkg/retry"

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

			tx, err := p.UnitOfWork.Begin(taskCtx)
			if err != nil {
				p.Logger.Error("Failed to begin transaction", "error", err)
				return
			}
			defer tx.Rollback(taskCtx)

			var usernames []string
			usernames, err = tx.Subscription().GetAllUniqueUsernames(taskCtx)
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
		shutdownErr := scheduler.Shutdown()
		if shutdownErr != nil {
			p.Logger.Error("Failed to shut down scheduler", "error", shutdownErr)
		}
	}()

	return nil
}

func (p *ParserImpl) runJobsWithAnts(ctx context.Context, usernames []string) {
	var wg sync.WaitGroup
	pool, _ := ants.NewPool(p.Config.Parser.StoryParsingPoolSize, ants.WithPreAlloc(true))
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
				//gosec:G404
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

	tx, err := p.UnitOfWork.Begin(ctx)
	if err != nil {
		return fmt.Errorf("failed to begin transaction for %s: %w", username, err)
	}
	defer tx.Rollback(ctx)

	newStories, err := p.filterNewStories(tx, stories)
	if err != nil {
		return fmt.Errorf("failed to filter new stories for %s: %w", username, err)
	}

	if len(newStories) == 0 {
		p.Logger.Info("No new stories for user", "username", username)
		return nil
	}

	p.Logger.Info("Found new stories", "username", username, "count", len(newStories))

	err = p.notifySubscribers(ctx, tx, username, newStories)
	if err != nil {
		return err
	}

	return tx.Commit(ctx)
}

func (p *ParserImpl) filterNewStories(tx unitofwork.Tx, stories []domain.StoryItem) ([]domain.StoryItem, error) {
	var newStories []domain.StoryItem
	for _, story := range stories {
		exists, err := p.checkStoryExists(tx, story.ID)
		if err != nil {
			p.Logger.Error("Failed to check story existence", "story_id", story.ID, "error", err)
			continue
		}
		if !exists {
			newStories = append(newStories, story)
		}
	}
	return newStories, nil
}

func (p *ParserImpl) notifySubscribers(ctx context.Context, tx unitofwork.Tx, username string, stories []domain.StoryItem) error {
	subscriberIDs, err := tx.Subscription().GetSubscribersForUser(ctx, username)
	if err != nil {
		return fmt.Errorf("failed to get subscribers for %s: %w", username, err)
	}

	if len(subscriberIDs) == 0 {
		p.Logger.Warn("Found new stories but no one is subscribed", "username", username)
		return nil
	}

	for _, story := range stories {
		if err := p.saveAndSendStory(ctx, tx, story, subscriberIDs); err != nil {
			p.Logger.Error("Failed to send story", "story_id", story.ID, "error", err)
		}
	}

	return nil
}

func (p *ParserImpl) saveAndSendStory(ctx context.Context, tx unitofwork.Tx, story domain.StoryItem, subscriberIDs []int64) error {
	dbStory := domain.Story{
		StoryID:   story.ID,
		UserName:  story.Username,
		CreatedAt: story.TakenAt,
	}
	err := tx.Story().Create(ctx, dbStory)
	if err != nil {
		if errors.Is(err, storyRepo.ErrCannotCreate) {
			p.Logger.Warn("Story might already exist or failed to create, skipping send", "story_id", dbStory.StoryID)
			return nil
		}
		p.Logger.Error("Failed to save story to DB", "story_id", dbStory.StoryID, "error", err)
		return err
	}

	for _, chatID := range subscriberIDs {
		sendMediaOperation := func() error {
			return p.Telegram.SendMediaByUrl(chatID, story.MediaURL)
		}
		err = retry.Do(ctx, p.Logger, "SendMediaByUrl", sendMediaOperation, retry.DefaultConfig())
		if err != nil {
			p.Logger.Error("Failed to send story to subscriber after retries",
				"chat_id", chatID, "url", story.MediaURL, "error", err)
		}
	}
	//gosec:G404
	time.Sleep(time.Duration(1500+rand.Intn(2000)) * time.Millisecond)
	return nil
}

func shuffleUsernames(usernames []string) []string {
	result := make([]string, len(usernames))
	copy(result, usernames)
	//gosec:G404
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

	var wg sync.WaitGroup
	var errsMutex sync.Mutex
	var errs []error
	var processed, skipped, failed int
	var statsMutex sync.Mutex

	p.Logger.Info("Processing stories", "count", len(stories))

	pool, _ := ants.NewPool(p.Config.Parser.StoryProcessingPoolSize, ants.WithPreAlloc(true))
	defer pool.Release()

	for _, story := range stories {
		wg.Add(1)
		storyToProcess := story
		err := pool.Submit(func() {
			defer wg.Done()
			err := p.processSingleStory(storyToProcess)
			if err != nil {
				errsMutex.Lock()
				errs = append(errs, err)
				errsMutex.Unlock()
				statsMutex.Lock()
				failed++
				statsMutex.Unlock()
			} else {
				statsMutex.Lock()
				processed++
				statsMutex.Unlock()
			}
		})
		if err != nil {
			wg.Done()
			p.Logger.Error("Failed to submit story processing job", "story_id", storyToProcess.ID, "error", err)
			statsMutex.Lock()
			failed++
			statsMutex.Unlock()
		}
	}

	wg.Wait()

	p.Logger.Info("Story processing completed", "total", len(stories),
		"processed", processed, "skipped", skipped, "failed", failed)

	if len(errs) > 0 {
		return fmt.Errorf("encountered %d errors during story parsing, first error: %w", len(errs), errs[0])
	}
	return nil
}

func (p *ParserImpl) processSingleStory(item domain.StoryItem) error {
	tx, err := p.UnitOfWork.Begin(context.Background())
	if err != nil {
		return fmt.Errorf("failed to begin transaction for story %s: %w", item.ID, err)
	}
	defer tx.Rollback(context.Background())

	exists, err := p.checkStoryExists(tx, item.ID)
	if err != nil {
		return fmt.Errorf("failed to check story existence: %w", err)
	}

	if exists {
		p.Logger.Debug("Story already processed", "storyID", item.ID)
		return nil
	}

	if err := p.processStoryItem(tx, item); err != nil {
		return fmt.Errorf("failed to process story %s: %w", item.ID, err)
	}

	return tx.Commit(context.Background())
}

func (p *ParserImpl) checkStoryExists(tx unitofwork.Tx, storyID string) (bool, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if storyID == "" {
		p.Logger.Warn("checkStoryExists called with empty storyID")
		return true, nil
	}

	_, err := tx.Story().GetByStoryID(ctx, storyID)
	if err != nil {
		if errors.Is(err, storyRepo.ErrNotFound) {
			return false, nil
		}
		return false, err
	}
	return true, nil
}

func (p *ParserImpl) processStoryItem(tx unitofwork.Tx, item domain.StoryItem) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	story := domain.Story{
		StoryID:   item.ID,
		UserName:  item.Username,
		CreatedAt: item.TakenAt,
	}

	if err := tx.Story().Create(ctx, story); err != nil {
		if errors.Is(err, storyRepo.ErrCannotCreate) {
			p.Logger.Warn("Story might already exist or failed to create, skipping send", "story_id", story.StoryID)
			return nil
		}
		return fmt.Errorf("failed to save story: %w", err)
	}

	p.Logger.Info("Processing media item", "username", item.Username, "url", item.MediaURL, "type", item.MediaType)
	p.Telegram.SendMediaToDefaultChannelByUrl(item.MediaURL)

	//gosec:G404
	delay := time.Duration(1500+rand.Intn(2000)) * time.Millisecond
	p.Logger.Info("Scheduled job: Waiting to avoid rate limit", "delay", delay)
	time.Sleep(delay)
	return nil
}

func (p *ParserImpl) SaveHighlight(highlight domain.Highlights) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	p.Logger.Info("Saving highlight", "username", highlight.UserName, "mediaURL", highlight.MediaURL)

	tx, err := p.UnitOfWork.Begin(ctx)
	if err != nil {
		return fmt.Errorf("failed to begin transaction for highlight: %w", err)
	}
	defer tx.Rollback(ctx)

	err = tx.Highlights().Create(ctx, highlight)
	if err != nil {
		return fmt.Errorf("failed to save highlight: %w", err)
	}
	return tx.Commit(ctx)
}

func (p *ParserImpl) SaveCurrentStory(currentStory domain.CurrentStory) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	p.Logger.Info("Saving current story", "username", currentStory.UserName, "mediaURL", currentStory.MediaURL)

	tx, err := p.UnitOfWork.Begin(ctx)
	if err != nil {
		return fmt.Errorf("failed to begin transaction for current story: %w", err)
	}
	defer tx.Rollback(ctx)

	err = tx.CurrentStory().Create(ctx, currentStory)
	if err != nil {
		return fmt.Errorf("failed to save current story: %w", err)
	}
	return tx.Commit(ctx)
}

func (p *ParserImpl) ClearCurrentStories(username string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	p.Logger.Info("Clearing current stories for user", "username", username)

	tx, err := p.UnitOfWork.Begin(ctx)
	if err != nil {
		return fmt.Errorf("failed to begin transaction for clearing stories: %w", err)
	}
	defer tx.Rollback(ctx)

	err = tx.CurrentStory().DeleteByUserName(ctx, username)
	if err != nil {
		return fmt.Errorf("failed to clear current stories for %s: %w", username, err)
	}
	return tx.Commit(ctx)
}
