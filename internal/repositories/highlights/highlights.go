package highlights

import (
	"context"
	"errors"
	"time"

	"github.com/orgball2608/insta-parser-telegram-bot/internal/domain"
)

type Highlights struct {
	ID        int
	UserName  string
	MediaURL  string
	CreatedAt time.Time
}

var ErrNotFound = errors.New("highlights not found")
var ErrCannotCreate = errors.New("error create highlights")

//go:generate go run go.uber.org/mock/mockgen -source=highlights.go -destination=mocks/mock.go

type Repository interface {
	GetByID(ctx context.Context, id int) (*domain.Highlights, error)
	GetByUserName(ctx context.Context, userName string) ([]*domain.Highlights, error)
	Create(ctx context.Context, highlights domain.Highlights) error
}
