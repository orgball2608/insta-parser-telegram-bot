package instagram

import (
	"context"
	"errors"

	"github.com/orgball2608/insta-parser-telegram-bot/internal/domain"
)

var ErrPrivateAccount = errors.New("account is private and cannot be accessed")

type HighlightReelProcessorFunc func(reel domain.HighlightReel) error

type Client interface {
	GetUserStories(userName string) ([]domain.StoryItem, error)
	GetUserHighlights(userName string, processorFunc HighlightReelProcessorFunc) error
	GetHighlightAlbumPreviews(userName string) ([]domain.HighlightAlbumPreview, error)
	GetSingleHighlightAlbum(userName, albumID string) (*domain.HighlightReel, error)
	GetUserPost(ctx context.Context, postURL string) (*domain.PostItem, error)
	GetUserReel(ctx context.Context, reelURL string) (*domain.PostItem, error)
}
