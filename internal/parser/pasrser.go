package parser

import "context"

type Client interface {
	ParseUserReelStories(ctx context.Context, username string) error
	ParseStories(ctx context.Context) error
}
