package paserimpl

import (
	"context"
	"errors"
	"fmt"
	"github.com/Davincible/goinsta/v3"
	"github.com/go-co-op/gocron/v2"
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/orgball2608/insta-parser-telegram-bot/internal/domain"
	storyRepo "github.com/orgball2608/insta-parser-telegram-bot/internal/repositories/story"
	"strings"
	"time"
)

func (p *ParserImpl) ScheduleParseStories(ctx context.Context) error {
	loc, _ := time.LoadLocation("Asia/Ho_Chi_Minh")
	s, err := gocron.NewScheduler(gocron.WithLocation(loc))
	if err != nil {
		return fmt.Errorf("new scheduler: %w", err)
	}

	_, err = s.NewJob(
		gocron.DurationRandomJob(
			time.Hour*1,
			time.Hour*24,
		),
		gocron.NewTask(
			func() {
				usernames := strings.Split(p.Config.Instagram.UsersParse, ";")
				for _, username := range usernames {
					err := p.ParseUserReelStories(ctx, username)
					if err != nil {
						p.Logger.Error("Parser error", "Error", err)
						p.Telegram.SendMessageToUser("parser error:" + err.Error())
					}
				}
			},
		),
	)
	if err != nil {
		return fmt.Errorf("start new job: %w", err)
	}

	s.Start()
	return nil
}

func (p *ParserImpl) ParseUserReelStories(ctx context.Context, username string) error {
	stories, err := p.Instagram.GetUserStories(username)
	if err != nil {
		p.Logger.Error("Error get user stories", "Error", err)
		return err
	}
	for _, story := range stories {
		storyID := story.GetID()

		p.Logger.Info("Parse story", "story id", storyID)

		createdAt := time.Unix(story.TakenAt, 0)
		_, err := p.StoryRepo.GetByStoryID(ctx, storyID)

		if err == nil {
			p.Logger.Info("Story already sent")
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

		mediaBytes := tgbotapi.FileBytes{
			Name:  "media",
			Bytes: media,
		}

		p.Telegram.SendMessageToChanel("New stories from " + username)

		imageUrl := story.Images.GetBest()

		p.Telegram.SendImageToChanelByUrl(imageUrl)

		if err := p.Telegram.SendFileToChannel(mediaBytes, story.MediaType); err != nil {
			p.Logger.Error("Error send media to channel", "Error", err)
			return err
		}

		p.Logger.Info("Media sent to channel")
	}
	return nil
}

func (p *ParserImpl) ParseStories(stories []*goinsta.Item) error {
	p.Logger.Info("Parse stories", "stories", len(stories))
	for _, story := range stories {
		storyID := story.GetID()

		p.Logger.Info("Parse story", "story id", storyID)

		media, err := story.Download()
		if err != nil {
			p.Logger.Error("Error download media", "Error", err)
			return err
		}

		mediaBytes := tgbotapi.FileBytes{
			Name:  "media",
			Bytes: media,
		}

		if err := p.Telegram.SendFileToChannel(mediaBytes, story.MediaType); err != nil {
			p.Logger.Error("Error send media to channel", "Error", err)
			return err
		}

		p.Logger.Info("Media sent to channel")
	}
	return nil
}
