package post

import (
	"context"
	"errors"

	"github.com/orgball2608/insta-parser-telegram-bot/internal/domain"
)

var (
	ErrAlreadyExists = errors.New("post parser already exists")
	ErrNotFound      = errors.New("post parser not found")
)

//go:generate go run go.uber.org/mock/mockgen -source=post.go -destination=mocks/mock.go
type Repository interface {
	// Create adds a new post parser entry
	Create(ctx context.Context, post domain.PostParser) error

	// GetByUsername returns all posts for a specific username
	GetByUsername(ctx context.Context, username string) ([]*domain.PostParser, error)

	// GetLatestByUsername returns the most recent posts for a specific username, limited by count
	GetLatestByUsername(ctx context.Context, username string, count int) ([]*domain.PostParser, error)

	// Exists checks if a post with the given ID already exists
	Exists(ctx context.Context, postID string) (bool, error)

	// CleanupOldRecords deletes records older than the specified duration
	CleanupOldRecords(ctx context.Context, olderThan string) (int64, error)
}
