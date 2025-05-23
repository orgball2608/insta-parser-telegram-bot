package pgx

import (
	"context"
	"fmt"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/orgball2608/insta-parser-telegram-bot/pkg/config"
	"github.com/orgball2608/insta-parser-telegram-bot/pkg/logger"
	"go.uber.org/fx"
)

// Opts holds dependencies for creating a pgx pool.
type Opts struct {
	fx.In
	LC     fx.Lifecycle
	Logger logger.Logger
	Config *config.Config
}

// New creates a new pgxpool.Pool and manages its lifecycle.
func New(opts Opts) (*pgxpool.Pool, error) {
	connStr := fmt.Sprintf(
		"postgres://%s:%s@%s:%d/%s?sslmode=%s",
		opts.Config.Postgres.User,
		opts.Config.Postgres.Pass,
		opts.Config.Postgres.Host,
		opts.Config.Postgres.Port,
		opts.Config.Postgres.Name,
		opts.Config.Postgres.SslMode,
	)

	pgx, err := pgxpool.New(context.Background(), connStr)
	if err != nil {
		return nil, fmt.Errorf("failed to create pgx pool: %w", err)
	}

	opts.LC.Append(
		fx.Hook{
			OnStart: func(ctx context.Context) error {
				if err := pgx.Ping(ctx); err != nil {
					return fmt.Errorf("failed to ping postgres: %w", err)
				}
				opts.Logger.Info("Connected to postgres")
				return nil
			},
			OnStop: func(ctx context.Context) error {
				pgx.Close()
				return nil
			},
		},
	)

	return pgx, nil
}
