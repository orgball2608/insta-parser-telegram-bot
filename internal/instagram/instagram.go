package instagram

import "github.com/Davincible/goinsta/v3"

type Client interface {
	Login() error
	ReloadSession() error
	GetUserStories(userName string) ([]*goinsta.Item, error)
	GetUserHighlights(userName string) ([]*goinsta.Reel, error)
	VisitProfile(username string) (*goinsta.User, error)
}
