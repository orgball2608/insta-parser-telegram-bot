package logger

import (
	"fmt"
	"io"
	"log/slog"
	"os"

	"github.com/getsentry/sentry-go"

	"github.com/rs/zerolog"
	slogmulti "github.com/samber/slog-multi"
	slogsentry "github.com/samber/slog-sentry/v2"
	slogzerolog "github.com/samber/slog-zerolog/v2"
)

type Impl struct {
	log *slog.Logger

	service string
	sentry  *sentry.Client
}

type Opts struct {
	Env string

	Sentry *sentry.Client
	Level  slog.Level
}

var _ Logger = (*Impl)(nil)

func New(opts Opts) *Impl {
	level := opts.Level

	var zeroLogWriter io.Writer
	if opts.Env == "production" {
		zeroLogWriter = os.Stderr
	} else {
		zeroLogWriter = zerolog.ConsoleWriter{Out: os.Stderr}
	}

	slogzerolog.SourceKey = "source"
	slogzerolog.ErrorKeys = []string{"error", "err"}

	zeroLogLogger := zerolog.New(zeroLogWriter)

	log := slog.New(
		slogmulti.Fanout(
			slogzerolog.Option{
				Level:     level,
				Logger:    &zeroLogLogger,
				AddSource: false,
			}.NewZerologHandler(),
			slogsentry.Option{Level: slog.LevelError, AddSource: true}.NewSentryHandler(),
		),
	)

	return &Impl{
		log:    log,
		sentry: opts.Sentry,
	}
}

func (c *Impl) Info(input string, fields ...any) {
	c.log.Info(input, fields...)
}

func (c *Impl) Warn(input string, fields ...any) {
	c.log.Warn(input, fields...)
}

func (c *Impl) Error(input string, fields ...any) {
	c.log.Error(input, fields...)
}

func (c *Impl) Debug(input string, fields ...any) {
	c.log.Debug(input, fields...)
}

func (c *Impl) WithComponent(name string) Logger {
	return &Impl{
		log:     c.log.With(slog.String("component", name)),
		sentry:  c.sentry,
		service: c.service,
	}
}

func (c *Impl) GetSlog() *slog.Logger {
	return c.log
}

// Printf implements fx.Printer interface.
func (c *Impl) Printf(format string, args ...interface{}) {
	c.Info(fmt.Sprintf(format, args...))
}
