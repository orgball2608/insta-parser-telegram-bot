package paserimpl

import (
	"context"
	"errors"
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/orgball2608/insta-parser-telegram-bot/internal/domain"
	storyRepo "github.com/orgball2608/insta-parser-telegram-bot/internal/repository/story"
	"time"
)

func (p *ParserImpl) ParseStories(ctx context.Context, username string) error {
	stories, err := p.Instagram.GetUserStories(username)
	if err != nil {
		p.Logger.Error("Error get user stories", "Error", err)
		return err
	}
	for _, story := range stories {
		storyID := story.GetID()

		p.Logger.Info("Story ID"+storyID, "Story ID")

		createdAt := time.Unix(story.TakenAt, 0)
		_, err := p.StoryRepo.GetByStoryID(ctx, storyID)

		if err == nil {
			p.Logger.Info("Stories already sent")
			continue
		}

		if errors.Is(err, storyRepo.ErrNotFound) {
			p.Logger.Info("Story not found in DB")
			story := domain.Story{
				StoryID:   storyID,
				UserName:  username,
				CreatedAt: createdAt,
			}
			if err := p.StoryRepo.Create(ctx, story); err != nil {
				p.Logger.Error("Error create story", "Error", err)
				return err
			}
		} else {
			p.Logger.Error("Error get story", "Error", err)
			return err
		}

		media, err := story.Download()
		if err != nil {
			p.Logger.Error("Error download media", "Error", err)
			return err
		}

		photoBytes := tgbotapi.FileBytes{
			Name:  "media",
			Bytes: media,
		}

		if err := p.Telegram.SendToChannel(photoBytes, story.MediaType, username); err != nil {
			p.Logger.Error("Error send media to channel", "Error", err)
			return err
		}

		p.Logger.Info("Media sent to channel")
	}
	return nil
}
