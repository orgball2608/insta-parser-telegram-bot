package fx

import (
	"github.com/orgball2608/insta-parser-telegram-bot/internal/repositories/currentstory"
	"github.com/orgball2608/insta-parser-telegram-bot/internal/repositories/highlights"
	"github.com/orgball2608/insta-parser-telegram-bot/internal/repositories/post"
	"github.com/orgball2608/insta-parser-telegram-bot/internal/repositories/story"
	"github.com/orgball2608/insta-parser-telegram-bot/internal/repositories/subscription"
	"go.uber.org/fx"
)

var Module = fx.Options(
	story.Module,
	highlights.Module,
	currentstory.Module,
	subscription.Module,
	post.Module,
)
