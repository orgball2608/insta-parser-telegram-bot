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
	"github.com/robfig/cron/v3"
	"go.uber.org/fx"
	"strings"
	"time"
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

func run(lc fx.Lifecycle, cfg *config.Config, logger logger.Logger, telegram telegram.Client, instagram instagram.Client, p parser.Client) {
	lc.Append(fx.Hook{
		OnStart: func(context.Context) error {
			ctx := context.Background()
			err := instagram.Login()
			if err != nil {
				logger.Error("Instagram login error", "Error", err)
				errString := err.Error()
				telegram.SendMessageToUser("Instagram login error:" + errString)
			}

			loc, _ := time.LoadLocation("Asia/Ho_Chi_Minh")
			c := cron.New(cron.WithLocation(loc))
			_, err = c.AddFunc("0 6,22 * * *", func() {
				usernames := strings.Split(cfg.Instagram.UserParse, ";")
				for _, username := range usernames {
					err := p.ParseStories(ctx, username)
					if err != nil {
						logger.Error("Parser error", "Error", err)
						errString := err.Error()
						telegram.SendMessageToUser("Parser error:" + errString)
					}
				}
			})
			if err != nil {
				return err
			}
			c.Start()

			return nil

			//go func() {
			//	for {
			//		currentTime := getCurrentTime()
			//		hour := currentTime.Hour()
			//		if 12 <= hour && hour <= 23 {
			//			usernames := strings.Split(cfg.Instagram.UserParse, ";")
			//			for _, username := range usernames {
			//				err := p.ParseStories(ctx, username)
			//				if err != nil {
			//					logger.Error("Parser error: %v", err)
			//					errString := err.Error()
			//					telegram.SendMessageToUser("Parser error:" + errString)
			//				}
			//				time.Sleep(time.Minute)
			//			}
			//		}
			//		time.Sleep(time.Minute * time.Duration(cfg.Parser.Minutes))
			//	}
			//}()
			//
			//return nil
		},
	})
}

//func getCurrentTime() time.Time {
//	now := time.Now()
//	loc, _ := time.LoadLocation("Asia/Ho_Chi_Minh")
//	return now.In(loc)
//}
