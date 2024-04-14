package parser

import (
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/orgball2608/insta-parser-telegram-bot/internal/config"
	"github.com/orgball2608/insta-parser-telegram-bot/internal/db"
	"github.com/orgball2608/insta-parser-telegram-bot/internal/instagram"
	"github.com/orgball2608/insta-parser-telegram-bot/internal/telegram"
	"log"
)

func Start(instagram *instagram.InstaUser, bot *telegram.Bot, postgres *db.Postgres, cfg *config.Config) error {
	stories, err := instagram.GetUserStories(cfg.Instagram.UserParse)
	if err != nil {
		return err
	}
	for _, stories := range stories {
		storiesId := stories.GetID()

		log.Println("Stories ID: ", storiesId)

		result := postgres.Check(storiesId)
		if result {
			log.Println("Stories already sent")
			continue
		}

		media, err := stories.Download()
		if err != nil {
			log.Printf("Error download media: %v", err)
			return err
		}

		photoBytes := tgbotapi.FileBytes{
			Name:  "media",
			Bytes: media,
		}

		if bot.SendToChannel(cfg.Telegram.Channel, photoBytes, stories.MediaType) != nil {
			log.Printf("Error send media to channel: %v", err)
			return err
		}

		log.Println("Media sent to channel")
	}
	return nil
}
