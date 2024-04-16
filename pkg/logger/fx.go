package logger

import (
	"github.com/orgball2608/insta-parser-telegram-bot/pkg/config"
	"go.uber.org/fx"
)

var FxOption = fx.Annotate(
	func(cfg *config.Config) *Impl {
		return New(
			Opts{
				Env: cfg.App.Env,
			},
		)
	},
	fx.As(new(Logger)),
)
