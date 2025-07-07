package unitofwork

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/orgball2608/insta-parser-telegram-bot/internal/repositories/currentstory"
	"github.com/orgball2608/insta-parser-telegram-bot/internal/repositories/highlights"
	"github.com/orgball2608/insta-parser-telegram-bot/internal/repositories/post"
	"github.com/orgball2608/insta-parser-telegram-bot/internal/repositories/story"
	"github.com/orgball2608/insta-parser-telegram-bot/internal/repositories/subscription"
	"github.com/orgball2608/insta-parser-telegram-bot/pkg/logger"
)

type UnitOfWork interface {
	Begin(ctx context.Context) (Tx, error)
}

type Tx interface {
	Commit(ctx context.Context) error
	Rollback(ctx context.Context) error
	Post() post.Repository
	Story() story.Repository
	Subscription() subscription.Repository
	Highlights() highlights.Repository
	CurrentStory() currentstory.Repository
}

type pgxUnitOfWork struct {
	pool   *pgxpool.Pool
	logger logger.Logger
}

func NewUnitOfWork(pool *pgxpool.Pool, logger logger.Logger) UnitOfWork {
	return &pgxUnitOfWork{
		pool:   pool,
		logger: logger,
	}
}

func (u *pgxUnitOfWork) Begin(ctx context.Context) (Tx, error) {
	tx, err := u.pool.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to begin transaction: %w", err)
	}
	return &pgxTx{
		tx:     tx,
		logger: u.logger,
	}, nil
}

type pgxTx struct {
	tx     pgx.Tx
	logger logger.Logger
}

func (t *pgxTx) Commit(ctx context.Context) error {
	return t.tx.Commit(ctx)
}

func (t *pgxTx) Rollback(ctx context.Context) error {
	return t.tx.Rollback(ctx)
}

func (t *pgxTx) Post() post.Repository {
	return post.NewPgx(t.tx, t.logger)
}

func (t *pgxTx) Story() story.Repository {
	return story.NewPgx(t.tx, t.logger)
}

func (t *pgxTx) Subscription() subscription.Repository {
	return subscription.NewPgxRepository(t.tx, t.logger)
}

func (t *pgxTx) Highlights() highlights.Repository {
	return highlights.NewPgxRepository(t.tx, t.logger)
}

func (t *pgxTx) CurrentStory() currentstory.Repository {
	return currentstory.NewPgxRepository(t.tx, t.logger)
}
