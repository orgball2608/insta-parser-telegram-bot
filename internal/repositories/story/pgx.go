package story

import (
	"context"
	"errors"
	"time"

	sq "github.com/Masterminds/squirrel"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/orgball2608/insta-parser-telegram-bot/internal/domain"
	"github.com/orgball2608/insta-parser-telegram-bot/internal/repositories"
	"github.com/orgball2608/insta-parser-telegram-bot/pkg/logger"
	"github.com/orgball2608/insta-parser-telegram-bot/pkg/retry"
)

func NewPgx(querier repositories.Querier, logger logger.Logger) *Pgx {
	return &Pgx{
		querier: querier,
		logger:  logger,
	}
}

var _ Repository = (*Pgx)(nil)

type Pgx struct {
	querier repositories.Querier
	logger  logger.Logger
}

func (p *Pgx) getStoryBy(ctx context.Context, cond sq.Eq, operationName string) (*domain.Story, error) {
	query, args, err := repositories.SqBuilder.
		Select("id", "story_id", "username", "created_at").
		From("story_parsers").
		Where(cond).ToSql()
	if err != nil {
		return nil, repositories.ErrBadQuery
	}

	story := Story{}
	queryOperation := func() error {
		return p.querier.QueryRow(ctx, query, args...).Scan(&story.ID, &story.StoryID, &story.UserName, &story.CreatedAt)
	}
	err = retry.Do(ctx, p.logger, operationName, queryOperation, retry.DefaultConfig())
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, err
	}

	return &domain.Story{
		ID:        story.ID,
		StoryID:   story.StoryID,
		UserName:  story.UserName,
		CreatedAt: story.CreatedAt,
	}, nil
}

func (p *Pgx) GetByID(ctx context.Context, id int) (*domain.Story, error) {
	return p.getStoryBy(ctx, sq.Eq{"id": id}, "GetStoryByID")
}

func (p *Pgx) GetByStoryID(ctx context.Context, storyID string) (*domain.Story, error) {
	return p.getStoryBy(ctx, sq.Eq{"story_id": storyID}, "GetByStoryID")
}

func (p *Pgx) Create(ctx context.Context, story domain.Story) error {
	query, args, err := repositories.SqBuilder.
		Insert("story_parsers").
		Columns(
			"story_id",
			"username",
			"created_at",
		).Values(
		story.StoryID,
		story.UserName,
		story.CreatedAt,
	).ToSql()
	if err != nil {
		return repositories.ErrBadQuery
	}

	execOperation := func() error {
		_, err := p.querier.Exec(ctx, query, args...)
		return err
	}
	err = retry.Do(ctx, p.logger, "CreateStory", execOperation, retry.DefaultConfig())
	if err != nil {
		return errors.Join(err, ErrCannotCreate)
	}

	return nil
}

func (p *Pgx) CleanupOldRecords(ctx context.Context, olderThan time.Duration) (int64, error) {
	cutoffTime := time.Now().Add(-olderThan)

	query, args, err := repositories.SqBuilder.
		Delete("story_parsers").
		Where(sq.Lt{"created_at": cutoffTime}).
		ToSql()
	if err != nil {
		return 0, repositories.ErrBadQuery
	}

	var result pgconn.CommandTag
	execOperation := func() error {
		var execErr error
		result, execErr = p.querier.Exec(ctx, query, args...)
		return execErr
	}
	err = retry.Do(ctx, p.logger, "CleanupOldRecords", execOperation, retry.DefaultConfig())
	if err != nil {
		return 0, err
	}

	return result.RowsAffected(), nil
}
