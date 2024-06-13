package parser

import (
	"context"
	"github.com/Davincible/goinsta/v3"
)

type Client interface {
	ParseUserReelStories(ctx context.Context, username string) error
	ScheduleParseStories(ctx context.Context) error
	ParseStories(stories []*goinsta.Item) error
}
