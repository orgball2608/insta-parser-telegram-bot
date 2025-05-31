package parser

import (
	"context"

	"github.com/Davincible/goinsta/v3"
	"github.com/orgball2608/insta-parser-telegram-bot/internal/domain"
)

type Client interface {
	ParseUserReelStories(ctx context.Context, username string) error
	ScheduleParseStories(ctx context.Context) error
	ParseStories(stories []*goinsta.Item) error
	SaveHighlight(highlight domain.Highlights) error
	SaveCurrentStory(currentStory domain.CurrentStory) error
	ClearCurrentStories(username string) error
}
