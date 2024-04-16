package instagram

import (
	"github.com/Davincible/goinsta/v3"
)

type Client interface {
	Login() error
	GetUserStories(userName string) ([]*goinsta.Item, error)
	ReloadSession() error
}
