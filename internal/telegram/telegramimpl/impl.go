package telegramimpl

import (
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/orgball2608/insta-parser-telegram-bot/internal/telegram"
	"github.com/orgball2608/insta-parser-telegram-bot/pkg/config"
	"github.com/orgball2608/insta-parser-telegram-bot/pkg/logger"
	"go.uber.org/fx"
)

type Opts struct {
	fx.In

	Config *config.Config
	Logger logger.Logger
}

type TelegramImpl struct {
	tgBot  *tgbotapi.BotAPI
	logger logger.Logger
	config *config.Config
}

func New(opts Opts) (*TelegramImpl, error) {
	tgBot, err := tgbotapi.NewBotAPI(opts.Config.Telegram.Token)
	if err != nil {
		opts.Logger.Error("Error creating bot", "Error", err)
		return nil, err
	}

	return &TelegramImpl{
		tgBot:  tgBot,
		logger: opts.Logger,
		config: opts.Config,
	}, nil
}

var _ telegram.Client = (*TelegramImpl)(nil)
