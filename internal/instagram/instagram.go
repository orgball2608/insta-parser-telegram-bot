package instagram

import (
	"github.com/orgball2608/insta-parser-telegram-bot/internal/domain"
)

type HighlightReelProcessorFunc func(reel domain.HighlightReel) error

type Client interface {
	GetUserStories(userName string) ([]domain.StoryItem, error)
	GetUserHighlights(userName string, processorFunc HighlightReelProcessorFunc) error
}
