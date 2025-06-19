package instagram

import (
	"errors"

	"github.com/orgball2608/insta-parser-telegram-bot/internal/domain"
)

var ErrPrivateAccount = errors.New("account is private and cannot be accessed")

type HighlightReelProcessorFunc func(reel domain.HighlightReel) error

type Client interface {
	GetUserStories(userName string) ([]domain.StoryItem, error)
	GetUserHighlights(userName string, processorFunc HighlightReelProcessorFunc) error
}
