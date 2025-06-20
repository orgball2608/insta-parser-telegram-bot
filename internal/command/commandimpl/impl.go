package commandimpl

import (
	"github.com/orgball2608/insta-parser-telegram-bot/internal/command"
	"github.com/orgball2608/insta-parser-telegram-bot/internal/instagram"
	"github.com/orgball2608/insta-parser-telegram-bot/internal/parser"
	"github.com/orgball2608/insta-parser-telegram-bot/internal/repositories/subscription"
	"github.com/orgball2608/insta-parser-telegram-bot/internal/telegram"
	"github.com/orgball2608/insta-parser-telegram-bot/pkg/config"
	"github.com/orgball2608/insta-parser-telegram-bot/pkg/logger"
	"go.uber.org/fx"
)

type Opts struct {
	fx.In

	Instagram        instagram.Client
	Telegram         telegram.Client
	Parser           parser.Client
	Logger           logger.Logger
	Config           *config.Config
	SubscriptionRepo subscription.Repository
}

type CommandImpl struct {
	Instagram        instagram.Client
	Telegram         telegram.Client
	Parser           parser.Client
	Logger           logger.Logger
	Config           *config.Config
	SubscriptionRepo subscription.Repository
}

func New(opts Opts) *CommandImpl {
	return &CommandImpl{
		Instagram:        opts.Instagram,
		Telegram:         opts.Telegram,
		Parser:           opts.Parser,
		Logger:           opts.Logger,
		Config:           opts.Config,
		SubscriptionRepo: opts.SubscriptionRepo,
	}
}

var _ command.Client = (*CommandImpl)(nil)
