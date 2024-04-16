package parser

import "context"

type Client interface {
	ParseStories(ctx context.Context, username string) error
}
