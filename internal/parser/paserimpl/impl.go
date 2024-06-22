package paserimpl

import (
	"github.com/orgball2608/insta-parser-telegram-bot/internal/instagram"
	"github.com/orgball2608/insta-parser-telegram-bot/internal/parser"
	"github.com/orgball2608/insta-parser-telegram-bot/internal/repositories/story"
	"github.com/orgball2608/insta-parser-telegram-bot/internal/telegram"
	"github.com/orgball2608/insta-parser-telegram-bot/pkg/config"
	"github.com/orgball2608/insta-parser-telegram-bot/pkg/logger"
	"go.uber.org/fx"
)

type ParserImplOpts struct {
	fx.In

	Instagram instagram.Client
	Telegram  telegram.Client
	StoryRepo story.Repository
	Logger    logger.Logger
	Config    *config.Config
}

type ParserImpl struct {
	Instagram instagram.Client
	Telegram  telegram.Client
	StoryRepo story.Repository
	Logger    logger.Logger
	Config    *config.Config
}

func New(opts ParserImplOpts) *ParserImpl {
	return &ParserImpl{
		Instagram: opts.Instagram,
		Telegram:  opts.Telegram,
		StoryRepo: opts.StoryRepo,
		Logger:    opts.Logger,
		Config:    opts.Config,
	}
}

var _ parser.Client = (*ParserImpl)(nil)
