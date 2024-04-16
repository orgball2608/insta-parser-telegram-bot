package paserimpl

import (
	"github.com/orgball2608/insta-parser-telegram-bot/internal/db"
	"github.com/orgball2608/insta-parser-telegram-bot/internal/instagram"
	"github.com/orgball2608/insta-parser-telegram-bot/internal/parser"
	"github.com/orgball2608/insta-parser-telegram-bot/internal/telegram"
	"github.com/orgball2608/insta-parser-telegram-bot/pkg/logger"
	"go.uber.org/fx"
)

type UserImplOpts struct {
	fx.In

	Instagram instagram.Client
	Telegram  telegram.Client
	Postgres  *db.Postgres
	Logger    logger.Logger
}

type ParserImpl struct {
	Instagram instagram.Client
	Telegram  telegram.Client
	Postgres  *db.Postgres
	Logger    logger.Logger
}

func NewParser(instagram instagram.Client, telegram telegram.Client, pg *db.Postgres, logger logger.Logger) *ParserImpl {
	return &ParserImpl{
		Instagram: instagram,
		Telegram:  telegram,
		Postgres:  pg,
		Logger:    logger,
	}
}

var _ parser.Client = (*ParserImpl)(nil)
