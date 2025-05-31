package logger

import (
	"github.com/getsentry/sentry-go"
	"github.com/orgball2608/insta-parser-telegram-bot/pkg/config"
	"go.uber.org/fx"
)

var FxOption = fx.Annotate(
	func(cfg *config.Config) *Impl {
		client, err := sentry.NewClient(sentry.ClientOptions{
			Dsn:              cfg.App.SentryUrl,
			TracesSampleRate: 1.0,
		})

		if err != nil {
			panic(err)
		}

		return New(
			Opts{
				Env:    cfg.App.Environment,
				Sentry: client,
			},
		)
	},
	fx.As(new(Logger)),
)
