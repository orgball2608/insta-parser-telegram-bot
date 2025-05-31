package fx

import (
	"github.com/orgball2608/insta-parser-telegram-bot/internal/repositories/currentstory"
	"github.com/orgball2608/insta-parser-telegram-bot/internal/repositories/highlights"
	"github.com/orgball2608/insta-parser-telegram-bot/internal/repositories/story"
	"go.uber.org/fx"
)

var Module = fx.Options(
	story.Module,
	highlights.Module,
	currentstory.Module,
)
