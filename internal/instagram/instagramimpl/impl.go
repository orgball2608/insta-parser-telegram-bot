package instagramimpl

import (
	"github.com/Davincible/goinsta/v3"
	"github.com/orgball2608/insta-parser-telegram-bot/internal/instagram"
	"github.com/orgball2608/insta-parser-telegram-bot/pkg/config"
	"github.com/orgball2608/insta-parser-telegram-bot/pkg/logger"
	"go.uber.org/fx"
)

type InstaImpl struct {
	client *goinsta.Instagram
	logger logger.Logger
}

type UserImplOpts struct {
	fx.In

	Config *config.Config
	Logger logger.Logger
}

func New(opts UserImplOpts) *InstaImpl {
	client := goinsta.New(opts.Config.Instagram.User, opts.Config.Instagram.Pass)

	return &InstaImpl{
		client: client,
		logger: opts.Logger,
	}
}

var _ instagram.Client = (*InstaImpl)(nil)
