package story

import (
	"context"
	"errors"
	"time"

	"github.com/orgball2608/insta-parser-telegram-bot/internal/domain"
)

type Story struct {
	ID        int
	StoryID   string
	UserName  string
	CreatedAt time.Time
}

var ErrNotFound = errors.New("story not found")
var ErrCannotCreate = errors.New("error create story")

//go:generate go run go.uber.org/mock/mockgen -source=story.go -destination=mocks/mock.go

type Repository interface {
	GetByID(ctx context.Context, id int) (*domain.Story, error)
	GetByStoryID(ctx context.Context, storyID string) (*domain.Story, error)
	Create(ctx context.Context, user domain.Story) error
	CleanupOldRecords(ctx context.Context, olderThan time.Duration) (int64, error)
}
