package retry

import (
	"context"
	"time"

	"github.com/cenkalti/backoff/v4"
	"github.com/orgball2608/insta-parser-telegram-bot/pkg/logger"
)

type Config struct {
	MaxRetries      uint64
	InitialInterval time.Duration
	MaxInterval     time.Duration
	Multiplier      float64
}

func DefaultConfig() Config {
	return Config{
		MaxRetries:      3,
		InitialInterval: 500 * time.Millisecond,
		MaxInterval:     5 * time.Second,
		Multiplier:      1.5,
	}
}

func Do(ctx context.Context, log logger.Logger, operationName string, operation func() error, cfg Config) error {
	bo := backoff.NewExponentialBackOff()
	bo.InitialInterval = cfg.InitialInterval
	bo.MaxInterval = cfg.MaxInterval
	bo.Multiplier = cfg.Multiplier
	bo.Reset()

	retryable := backoff.WithMaxRetries(bo, cfg.MaxRetries)
	retryableWithContext := backoff.WithContext(retryable, ctx)

	notify := func(err error, t time.Duration) {
		log.Warn(
			"Operation failed, retrying...",
			"operation", operationName,
			"error", err,
			"next_attempt_in", t.Round(time.Millisecond).String(),
		)
	}

	return backoff.RetryNotify(operation, retryableWithContext, notify)
}
