package subscription

import (
	"context"
	"errors"

	"github.com/orgball2608/insta-parser-telegram-bot/internal/domain"
)

var (
	ErrAlreadyExists = errors.New("subscription already exists")
	ErrNotFound      = errors.New("subscription not found")
)

//go:generate go run go.uber.org/mock/mockgen -source=subscription.go -destination=mocks/mock.go
type Repository interface {
	Create(ctx context.Context, sub domain.Subscription) error
	Delete(ctx context.Context, chatID int64, username string) error
	GetByChatID(ctx context.Context, chatID int64) ([]*domain.Subscription, error)
	GetAllUniqueUsernames(ctx context.Context) ([]string, error)
	GetSubscribersForUser(ctx context.Context, username string) ([]int64, error)

	// New methods for subscription types
	GetSubscribersForUserByType(ctx context.Context, username string, subscriptionType string) ([]int64, error)
	GetAllUniqueUsernamesByType(ctx context.Context, subscriptionType string) ([]string, error)
	UpdateSubscriptionType(ctx context.Context, chatID int64, username string, subscriptionType string) error
}
