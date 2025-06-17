package instagram

import (
	"github.com/orgball2608/insta-parser-telegram-bot/internal/domain"
)

type Client interface {
	GetUserStories(userName string) ([]domain.StoryItem, error)
	GetUserHighlights(userName string) ([]domain.HighlightReel, error)
}
