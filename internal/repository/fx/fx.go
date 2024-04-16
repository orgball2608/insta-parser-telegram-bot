package fx

import (
	"github.com/orgball2608/insta-parser-telegram-bot/internal/repository/story"
	"go.uber.org/fx"
)

var Module = fx.Options(
	story.Module,
)
