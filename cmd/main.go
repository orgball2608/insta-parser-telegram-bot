package main

import (
	"context"
	"github.com/orgball2608/insta-parser-telegram-bot/internal/instagram"
	"github.com/orgball2608/insta-parser-telegram-bot/internal/instagram/instagramimpl"
	"github.com/orgball2608/insta-parser-telegram-bot/internal/parser"
	"github.com/orgball2608/insta-parser-telegram-bot/internal/parser/paserimpl"
	"github.com/orgball2608/insta-parser-telegram-bot/internal/pgx"
	repositories "github.com/orgball2608/insta-parser-telegram-bot/internal/repository/fx"
	"github.com/orgball2608/insta-parser-telegram-bot/internal/telegram"
	"github.com/orgball2608/insta-parser-telegram-bot/internal/telegram/telegramimpl"
	"github.com/orgball2608/insta-parser-telegram-bot/pkg/config"
	"github.com/orgball2608/insta-parser-telegram-bot/pkg/logger"
	"go.uber.org/fx"
)

func main() {
	fx.New(
		fx.Provide(
			config.NewConfig,
			logger.FxOption,
			fx.Annotate(
				telegramimpl.NewBot,
				fx.As(new(telegram.Client)),
			),
			fx.Annotate(
				instagramimpl.NewUser,
				fx.As(new(instagram.Client)),
			),
			pgx.New,
			fx.Annotate(
				paserimpl.NewParser,
				fx.As(new(parser.Client)),
			),
		),
		repositories.Module,
		fx.Invoke(pgx.New),
		fx.Invoke(run),
	).Run()
}

func run(lc fx.Lifecycle, logger logger.Logger, telegram telegram.Client, instagram instagram.Client, p parser.Client) {
	lc.Append(fx.Hook{
		OnStart: func(context.Context) error {
			ctx := context.Background()
			err := instagram.Login()
			if err != nil {
				logger.Error("Instagram login error", "Error", err)
				telegram.SendMessageToUser("Instagram login error:" + err.Error())
			}

			err = p.ParseStories(ctx)
			if err != nil {
				logger.Error("Parse stories error", "Error", err)
				telegram.SendMessageToUser("Parse stories error:" + err.Error())
			}

			return nil
		},
	})
}
