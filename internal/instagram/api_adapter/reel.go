package api_adapter

import (
	"context"

	"github.com/orgball2608/insta-parser-telegram-bot/internal/domain"
)

func (a *APIAdapter) GetUserReel(ctx context.Context, reelURL string) (*domain.PostItem, error) {
	return a.scrapeMedia(ctx, reelURL, "reel")
}
