package paserimpl

import (
	"context"
	"errors"
	"fmt"
	"math/rand"
	"strings"
	"sync"
	"time"

	"github.com/go-co-op/gocron/v2"
	"github.com/orgball2608/insta-parser-telegram-bot/internal/domain"
	storyRepo "github.com/orgball2608/insta-parser-telegram-bot/internal/repositories/story"
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
		gocron.DurationRandomJob(time.Hour*3, time.Hour*24*2),
		gocron.NewTask(func() {
			if ctx.Err() != nil {
				p.Logger.Info("Context cancelled, stopping story parsing schedule")
				return
			}

			taskCtx, cancel := context.WithTimeout(ctx, 15*time.Minute)
			defer cancel()

			usernames := strings.Split(p.Config.Instagram.UsersParse, ";")
			p.Logger.Info("Starting scheduled story parsing", "userCount", len(usernames))
			shuffledUsernames := shuffleUsernames(usernames)

			for i, username := range shuffledUsernames {
				username = strings.TrimSpace(username)
				if username == "" {
					continue
				}

				if i > 0 {
					delay := time.Duration(10+rand.Intn(35)) * time.Second
					p.Logger.Info("Waiting before processing next user", "delay", delay.String(), "nextUsername", username)
					select {
					case <-taskCtx.Done():
						p.Logger.Warn("Context cancelled during delay between users")
						return
					case <-time.After(delay):
					}
				}

				p.Logger.Info("Parsing stories for user", "username", username)
				if err := p.ParseUserStories(taskCtx, username); err != nil {
					p.Logger.Error("Failed to parse stories for user", "username", username, "error", err)
					time.Sleep(time.Duration(2+rand.Intn(3)) * time.Second)

					p.Telegram.SendMessageToDefaultChannel(fmt.Sprintf("Failed to parse stories for @%s: %s", username, err.Error()))
				} else {
					p.Logger.Info("Successfully parsed stories for user", "username", username)
				}

				jitter := time.Duration(rand.Intn(10)) * time.Second
				pauseDuration := 8*time.Second + jitter
				select {
				case <-taskCtx.Done():
					p.Logger.Warn("Context cancelled during story parsing")
					return
				case <-time.After(pauseDuration):
				}
			}
			p.Logger.Info("Completed scheduled story parsing")
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

func shuffleUsernames(usernames []string) []string {
	result := make([]string, len(usernames))
	copy(result, usernames)
	rand.New(rand.NewSource(time.Now().UnixNano()))
	for i := len(result) - 1; i > 0; i-- {
		j := rand.Intn(i + 1)
		result[i], result[j] = result[j], result[i]
	}
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
		if errors.Is(err, storyRepo.ErrNotFound) {
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
