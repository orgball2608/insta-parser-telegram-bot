package paserimpl

import (
	"github.com/orgball2608/insta-parser-telegram-bot/internal/instagram"
	"github.com/orgball2608/insta-parser-telegram-bot/internal/parser"
	"github.com/orgball2608/insta-parser-telegram-bot/internal/repository/story"
	"github.com/orgball2608/insta-parser-telegram-bot/internal/telegram"
	"github.com/orgball2608/insta-parser-telegram-bot/pkg/config"
	"github.com/orgball2608/insta-parser-telegram-bot/pkg/logger"
	"go.uber.org/fx"
)

type UserImplOpts struct {
	fx.In

	Instagram instagram.Client
	Telegram  telegram.Client
	StoryRepo story.Repository
	Logger    logger.Logger
	config    *config.Config
}

type ParserImpl struct {
	Instagram instagram.Client
	Telegram  telegram.Client
	StoryRepo story.Repository
	Logger    logger.Logger
	Config    *config.Config
}

func NewParser(instagram instagram.Client, telegram telegram.Client, storyRepo story.Repository, logger logger.Logger, config *config.Config) *ParserImpl {
	return &ParserImpl{
		Instagram: instagram,
		Telegram:  telegram,
		StoryRepo: storyRepo,
		Logger:    logger,
		Config:    config,
	}
}

var _ parser.Client = (*ParserImpl)(nil)
