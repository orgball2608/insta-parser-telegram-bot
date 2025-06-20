package commandimpl

import (
	"context"
	"fmt"
	"strconv"
	"time"

	"github.com/orgball2608/insta-parser-telegram-bot/internal/command"
	"github.com/orgball2608/insta-parser-telegram-bot/internal/instagram"
	"github.com/orgball2608/insta-parser-telegram-bot/internal/parser"
	"github.com/orgball2608/insta-parser-telegram-bot/internal/repositories/subscription"
	"github.com/orgball2608/insta-parser-telegram-bot/internal/telegram"
	"github.com/orgball2608/insta-parser-telegram-bot/pkg/config"
	"github.com/orgball2608/insta-parser-telegram-bot/pkg/logger"
	"github.com/orgball2608/insta-parser-telegram-bot/pkg/retry"
	"go.uber.org/fx"
)

type Opts struct {
	fx.In

	Instagram        instagram.Client
	Telegram         telegram.Client
	Parser           parser.Client
	Logger           logger.Logger
	Config           *config.Config
	SubscriptionRepo subscription.Repository
}

type CommandImpl struct {
	Instagram        instagram.Client
	Telegram         telegram.Client
	Parser           parser.Client
	Logger           logger.Logger
	Config           *config.Config
	SubscriptionRepo subscription.Repository
}

func New(opts Opts) *CommandImpl {
	return &CommandImpl{
		Instagram:        opts.Instagram,
		Telegram:         opts.Telegram,
		Parser:           opts.Parser,
		Logger:           opts.Logger,
		Config:           opts.Config,
		SubscriptionRepo: opts.SubscriptionRepo,
	}
}

var _ command.Client = (*CommandImpl)(nil)

// formatNumber converts an integer to a string with commas as thousands separators.
// Example: 1234567 -> "1,234,567"
func formatNumber(n int) string {
	s := strconv.Itoa(n)
	if n < 0 {
		s = s[1:]
	}

	le := len(s)
	if le <= 3 {
		if n < 0 {
			return "-" + s
		}
		return s
	}

	sepCount := (le - 1) / 3

	res := make([]byte, le+sepCount)

	j := len(res) - 1
	for i := le - 1; i >= 0; i-- {
		res[j] = s[i]
		j--
		if (le-i)%3 == 0 && i > 0 {
			res[j] = ','
			j--
		}
	}

	if n < 0 {
		return "-" + string(res)
	}
	return string(res)
}

func (c *CommandImpl) doWithRetryNotify(
	ctx context.Context,
	chatID int64,
	messageID int,
	initialMessage string,
	operationName string,
	operation func() error,
) error {
	var attempt int
	notifyFunc := func(err error, d time.Duration) {
		attempt++
		c.Logger.Warn(
			"Operation failed, retrying...",
			"operation", operationName,
			"attempt", attempt,
			"error", err,
			"next_attempt_in", d.Round(time.Millisecond).String(),
		)

		c.Telegram.EditMessageText(
			chatID,
			messageID,
			fmt.Sprintf("%s\n\n_Operation failed, retrying... (Attempt %d)_", initialMessage, attempt),
		)
	}

	return retry.RetryWithCustomNotify(ctx, operationName, operation, retry.DefaultConfig(), notifyFunc)
}
