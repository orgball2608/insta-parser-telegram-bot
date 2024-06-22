package main

import (
	"github.com/orgball2608/insta-parser-telegram-bot/internal/app"
	"go.uber.org/fx"
)

func main() {
	fx.New(
		app.App,
	).Run()
}
