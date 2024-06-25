package instagramimpl

import (
	"github.com/Davincible/goinsta/v3"
	"github.com/orgball2608/insta-parser-telegram-bot/internal/instagram"
	"github.com/orgball2608/insta-parser-telegram-bot/pkg/config"
	"github.com/orgball2608/insta-parser-telegram-bot/pkg/logger"
	"go.uber.org/fx"
)

type IgImpl struct {
	Client *goinsta.Instagram
	Logger logger.Logger
	Config *config.Config
}

type Opts struct {
	fx.In

	Config *config.Config
	Logger logger.Logger
}

func New(opts Opts) *IgImpl {
	client := goinsta.New(opts.Config.Instagram.User, opts.Config.Instagram.Pass)

	return &IgImpl{
		Client: client,
		Logger: opts.Logger,
		Config: opts.Config,
	}
}

var _ instagram.Client = (*IgImpl)(nil)
