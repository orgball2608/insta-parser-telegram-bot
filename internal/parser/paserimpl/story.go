package paserimpl

import (
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"log"
	"time"
)

func (p *ParserImpl) ParseStories(username string) error {
	stories, err := p.Instagram.GetUserStories(username)
	if err != nil {
		return err
	}
	for _, stories := range stories {
		storiesId := stories.GetID()

		log.Println("Stories ID: ", storiesId)

		takenAt := time.Unix(stories.TakenAt, 0)
		result := p.Postgres.Check(storiesId, username, takenAt)
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

		if p.Telegram.SendToChannel(photoBytes, stories.MediaType, username) != nil {
			log.Printf("Error send media to channel: %v", err)
			return err
		}

		log.Println("Media sent to channel")
	}
	return nil
}
