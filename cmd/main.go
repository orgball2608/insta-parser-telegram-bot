package main

import (
	"context"
	"github.com/orgball2608/insta-parser-telegram-bot/internal/instagram"
	"github.com/orgball2608/insta-parser-telegram-bot/internal/instagram/instagramimpl"
	"github.com/orgball2608/insta-parser-telegram-bot/internal/parser/paserimpl"
	"github.com/orgball2608/insta-parser-telegram-bot/internal/telegram/telegramimpl"
	"github.com/orgball2608/insta-parser-telegram-bot/pkg/config"
	"github.com/orgball2608/insta-parser-telegram-bot/pkg/logger"
	"strings"
	"time"

	"github.com/orgball2608/insta-parser-telegram-bot/internal/db"
	"github.com/orgball2608/insta-parser-telegram-bot/internal/parser"
	"github.com/orgball2608/insta-parser-telegram-bot/internal/telegram"
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
			db.NewConnect,
			fx.Annotate(
				instagramimpl.NewUser,
				fx.As(new(instagram.Client)),
			),
			fx.Annotate(
				paserimpl.NewParser,
				fx.As(new(parser.Client)),
			),
		),
		fx.Invoke(run),
	).Run()
}

func run(lc fx.Lifecycle, cfg *config.Config, logger logger.Logger, telegram telegram.Client,
	pg *db.Postgres, instagram instagram.Client, p parser.Client) {
	lc.Append(fx.Hook{
		OnStart: func(context.Context) error {
			err := pg.MigrationInit()
			if err != nil {
				logger.Error("Migration error: %v", err)
				errString := err.Error()
				telegram.SendMessageToUser("Migration error:" + errString)
			}

			err = instagram.Login()
			if err != nil {
				logger.Error("Instagram login error: %v", err)
				errString := err.Error()
				telegram.SendMessageToUser("Instagram login error:" + errString)
			}

			go func() {
				for {
					currentTime := getCurrentTime()
					hour := currentTime.Hour()
					if 12 <= hour && hour <= 14 {
						usernames := strings.Split(cfg.Instagram.UserParse, ";")
						for _, username := range usernames {
							err := p.ParseStories(username)
							if err != nil {
								logger.Error("Parser error: %v", err)
								errString := err.Error()
								telegram.SendMessageToUser("Parser error:" + errString)
							}
							time.Sleep(time.Minute)
						}
					}
					time.Sleep(time.Minute * time.Duration(cfg.Parser.Minutes))
				}
			}()

			return nil
		},
	})
}

func getCurrentTime() time.Time {
	now := time.Now()
	loc, _ := time.LoadLocation("Asia/Ho_Chi_Minh")
	return now.In(loc)
}
