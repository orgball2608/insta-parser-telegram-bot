package instagram

import (
	"errors"
	"github.com/Davincible/goinsta/v3"
	"github.com/orgball2608/insta-parser-telegram-bot/internal/config"
	"log"
)

type InstaUser struct {
	User *goinsta.Instagram
}

func NewUser(username, password string) *InstaUser {
	user := goinsta.New(username, password)

	return &InstaUser{
		User: user,
	}
}

func (i *InstaUser) LoginInstagram(cfg *config.Config) error {
	log.Printf("User: %s, Pass: %s\n", cfg.Instagram.User, cfg.Instagram.Pass)
	if err := i.ReloadSession(cfg); err != nil {
		err := i.User.Login()
		if err != nil {
			log.Printf("Login error: %v", err)
			return err
		}

		log.Println("Successfully logged in by login and password")

		if err := i.User.Export("./goinsta-session"); err != nil {
			log.Printf("Couldn't save the session: %v", err)
		}
	}

	log.Println("Login by session success")
	return nil
}

func (i *InstaUser) GetUserStories(userName string) ([]*goinsta.Item, error) {
	log.Println("userName: ", userName)
	profile, err := i.User.VisitProfile(userName)
	if err != nil {
		log.Printf("VisitProfile error: %v", err)
		return nil, err
	}

	stories := profile.Stories.Reel
	if err != nil {
		log.Printf("Stories error: %v", err)
		return nil, err
	}
	log.Printf("Stories: %v\n", stories.Items)

	return stories.Items, nil
}

func (i *InstaUser) ReloadSession(cfg *config.Config) error {
	insta, err := goinsta.Import("./goinsta-session")
	if err != nil {
		log.Printf("Couldn't recover the session: %v", err)
		return errors.New("couldn't recover the session")
	}

	i.User = insta

	log.Println("Successfully logged in by session")
	return nil
}
