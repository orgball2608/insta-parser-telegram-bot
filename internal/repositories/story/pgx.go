package story

import (
	"context"
	"errors"
	"time"

	sq "github.com/Masterminds/squirrel"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/orgball2608/insta-parser-telegram-bot/internal/domain"
	"github.com/orgball2608/insta-parser-telegram-bot/internal/repositories"
)

func NewPgx(pg *pgxpool.Pool) *Pgx {
	return &Pgx{
		pg: pg,
	}
}

var _ Repository = (*Pgx)(nil)

type Pgx struct {
	pg *pgxpool.Pool
}

func (p *Pgx) GetByID(ctx context.Context, id int) (*domain.Story, error) {
	query, args, err := repositories.SqBuilder.
		Select("id", "story_id", "username", "created_at").
		From("story_parsers").
		Where(
			sq.Eq{"id": id},
		).ToSql()
	if err != nil {
		return nil, repositories.ErrBadQuery
	}

	story := Story{}
	err = p.pg.QueryRow(ctx, query, args...).Scan(&story.ID, &story.StoryID, &story.UserName, &story.CreatedAt)
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

func (p *Pgx) GetByStoryID(ctx context.Context, storyID string) (*domain.Story, error) {
	query, args, err := repositories.SqBuilder.
		Select("id", "story_id", "username", "created_at").
		From("story_parsers").
		Where(
			sq.Eq{"story_id": storyID},
		).ToSql()
	if err != nil {
		return nil, repositories.ErrBadQuery
	}

	story := Story{}
	err = p.pg.QueryRow(ctx, query, args...).Scan(&story.ID, &story.StoryID, &story.UserName, &story.CreatedAt)
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

	_, err = p.pg.Exec(ctx, query, args...)
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

	result, err := p.pg.Exec(ctx, query, args...)
	if err != nil {
		return 0, err
	}

	return result.RowsAffected(), nil
}
