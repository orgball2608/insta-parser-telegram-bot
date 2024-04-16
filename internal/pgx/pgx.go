package pgx

import (
	"context"
	"fmt"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/orgball2608/insta-parser-telegram-bot/pkg/config"
	"github.com/orgball2608/insta-parser-telegram-bot/pkg/logger"
	"go.uber.org/fx"
)

type Opts struct {
	fx.In
	LC fx.Lifecycle

	Logger logger.Logger
	Config *config.Config
}

func New(opts Opts) (*pgxpool.Pool, error) {
	pgx, err := pgxpool.New(
		context.Background(),
		fmt.Sprintf("dbname=%s user=%s password=%s host=%s port=%d sslmode=%s ", opts.Config.Postgres.Name, opts.Config.Postgres.User, opts.Config.Postgres.Pass, opts.Config.Postgres.Host, opts.Config.Postgres.Port, opts.Config.Postgres.SslMode),
	)
	if err != nil {
		return nil, err
	}

	opts.LC.Append(
		fx.Hook{
			OnStop: func(ctx context.Context) error {
				pgx.Close()
				return nil
			},
			OnStart: func(ctx context.Context) error {
				err := pgx.Ping(ctx)
				if err != nil {
					return err
				}

				opts.Logger.Info("Connected to postgres")
				return nil
			},
		},
	)

	return pgx, nil
}
