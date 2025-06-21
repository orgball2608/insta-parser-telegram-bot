package paserimpl

import (
	"context"
	"fmt"
	"time"

	"github.com/go-co-op/gocron/v2"
	"github.com/orgball2608/insta-parser-telegram-bot/internal/instagram"
	"github.com/orgball2608/insta-parser-telegram-bot/internal/parser"
	"github.com/orgball2608/insta-parser-telegram-bot/internal/repositories/currentstory"
	"github.com/orgball2608/insta-parser-telegram-bot/internal/repositories/highlights"
	"github.com/orgball2608/insta-parser-telegram-bot/internal/repositories/post"
	"github.com/orgball2608/insta-parser-telegram-bot/internal/repositories/story"
	"github.com/orgball2608/insta-parser-telegram-bot/internal/repositories/subscription"
	"github.com/orgball2608/insta-parser-telegram-bot/internal/telegram"
	"github.com/orgball2608/insta-parser-telegram-bot/pkg/config"
	"github.com/orgball2608/insta-parser-telegram-bot/pkg/logger"
	"go.uber.org/fx"
)

type Opts struct {
	fx.In

	Instagram        instagram.Client
	Telegram         telegram.Client
	StoryRepo        story.Repository
	HighlightsRepo   highlights.Repository
	CurrentStoryRepo currentstory.Repository
	PostRepo         post.Repository
	Logger           logger.Logger
	Config           *config.Config
	SubscriptionRepo subscription.Repository
}

type ParserImpl struct {
	Instagram        instagram.Client
	Telegram         telegram.Client
	StoryRepo        story.Repository
	HighlightsRepo   highlights.Repository
	CurrentStoryRepo currentstory.Repository
	PostRepo         post.Repository
	Logger           logger.Logger
	Config           *config.Config
	SubscriptionRepo subscription.Repository
	Scheduler        gocron.Scheduler
}

func New(opts Opts) *ParserImpl {
	loc, err := time.LoadLocation("Asia/Ho_Chi_Minh")
	if err != nil {
		loc = time.Local
		opts.Logger.Warn("Failed to load Asia/Ho_Chi_Minh timezone, using local timezone", "error", err)
	}

	scheduler, err := gocron.NewScheduler(gocron.WithLocation(loc))
	if err != nil {
		opts.Logger.Error("Failed to create scheduler", "error", err)
	}

	return &ParserImpl{
		Instagram:        opts.Instagram,
		Telegram:         opts.Telegram,
		StoryRepo:        opts.StoryRepo,
		HighlightsRepo:   opts.HighlightsRepo,
		CurrentStoryRepo: opts.CurrentStoryRepo,
		PostRepo:         opts.PostRepo,
		Logger:           opts.Logger,
		Config:           opts.Config,
		SubscriptionRepo: opts.SubscriptionRepo,
		Scheduler:        scheduler,
	}
}

var _ parser.Client = (*ParserImpl)(nil)

func (p *ParserImpl) ScheduleDatabaseCleanup(ctx context.Context) error {
	loc, err := time.LoadLocation("Asia/Ho_Chi_Minh")
	if err != nil {
		loc = time.Local
		p.Logger.Warn("Failed to load Asia/Ho_Chi_Minh timezone, using local timezone", "error", err)
	}

	scheduler, err := gocron.NewScheduler(gocron.WithLocation(loc))
	if err != nil {
		return fmt.Errorf("failed to create cleanup scheduler: %w", err)
	}

	_, err = scheduler.NewJob(
		gocron.DailyJob(
			1,
			gocron.NewAtTimes(gocron.NewAtTime(3, 0, 0)),
		),
		gocron.NewTask(func() {
			if ctx.Err() != nil {
				p.Logger.Info("Context cancelled, stopping database cleanup job")
				return
			}

			p.Logger.Info("Starting scheduled database cleanup job")

			cleanupCtx, cancel := context.WithTimeout(ctx, 5*time.Minute)
			defer cancel()

			const cleanupDuration = 5 * 24 * time.Hour

			rowsDeleted, err := p.StoryRepo.CleanupOldRecords(cleanupCtx, cleanupDuration)
			if err != nil {
				p.Logger.Error("Failed to clean up old records", "error", err)
				return
			}

			p.Logger.Info("Database cleanup completed successfully", "rows_deleted", rowsDeleted)
		}),
	)

	if err != nil {
		return fmt.Errorf("failed to schedule database cleanup: %w", err)
	}

	scheduler.Start()

	go func() {
		<-ctx.Done()
		p.Logger.Info("Stopping database cleanup scheduler")
		if err := scheduler.Shutdown(); err != nil {
			p.Logger.Error("Failed to shut down cleanup scheduler", "error", err)
		}
	}()

	return nil
}
