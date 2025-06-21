package parser

import (
	"context"

	"github.com/orgball2608/insta-parser-telegram-bot/internal/domain"
)

type Client interface {
	ParseUserStories(ctx context.Context, username string) error
	ScheduleParseStories(ctx context.Context) error
	ProcessStories(stories []domain.StoryItem) error
	SaveHighlight(highlight domain.Highlights) error
	SaveCurrentStory(currentStory domain.CurrentStory) error
	ClearCurrentStories(username string) error
	ScheduleDatabaseCleanup(ctx context.Context) error
	SchedulePostChecking(ctx context.Context) error
}
