package currentstory

import (
	"context"
	"errors"
	"time"

	"github.com/orgball2608/insta-parser-telegram-bot/internal/domain"
)

type CurrentStory struct {
	ID        int
	UserName  string
	MediaURL  string
	CreatedAt time.Time
}

var ErrNotFound = errors.New("current story not found")
var ErrCannotCreate = errors.New("error create current story")

//go:generate go run go.uber.org/mock/mockgen -source=currentstory.go -destination=mocks/mock.go

type Repository interface {
	GetByID(ctx context.Context, id int) (*domain.CurrentStory, error)
	GetByUserName(ctx context.Context, userName string) ([]*domain.CurrentStory, error)
	Create(ctx context.Context, currentStory domain.CurrentStory) error
	DeleteByUserName(ctx context.Context, userName string) error
}
