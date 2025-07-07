package post

import (
	"context"
	"errors"
	"time"

	"github.com/jackc/pgx/v5/pgconn"
	"github.com/orgball2608/insta-parser-telegram-bot/internal/domain"
	"github.com/orgball2608/insta-parser-telegram-bot/internal/repositories"
	"github.com/orgball2608/insta-parser-telegram-bot/pkg/logger"
	"github.com/orgball2608/insta-parser-telegram-bot/pkg/retry"

	sq "github.com/Masterminds/squirrel"
)

type Pgx struct {
	querier repositories.Querier
	logger  logger.Logger
}

func NewPgx(querier repositories.Querier, logger logger.Logger) *Pgx {
	return &Pgx{
		querier: querier,
		logger:  logger.WithComponent("PostParserRepo"),
	}
}

var _ Repository = (*Pgx)(nil)

func (p *Pgx) Create(ctx context.Context, post domain.PostParser) error {
	query, args, err := repositories.SqBuilder.
		Insert("post_parsers").
		Columns("post_id", "username", "post_url", "created_at").
		Values(post.PostID, post.Username, post.PostURL, time.Now()).
		ToSql()
	if err != nil {
		return repositories.ErrBadQuery
	}

	execOperation := func() error {
		_, err := p.querier.Exec(ctx, query, args...)
		return err
	}

	err = retry.Do(ctx, p.logger, "CreatePostParser", execOperation, retry.DefaultConfig())
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			return ErrAlreadyExists
		}
		return err
	}
	return nil
}

func (p *Pgx) GetByUsername(ctx context.Context, username string) ([]*domain.PostParser, error) {
	query, args, err := repositories.SqBuilder.
		Select("id", "post_id", "username", "post_url", "created_at").
		From("post_parsers").
		Where(sq.Eq{"username": username}).
		OrderBy("created_at DESC").
		ToSql()
	if err != nil {
		return nil, repositories.ErrBadQuery
	}

	rows, err := p.querier.Query(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var posts []*domain.PostParser
	for rows.Next() {
		var post domain.PostParser
		if err := rows.Scan(&post.ID, &post.PostID, &post.Username, &post.PostURL, &post.CreatedAt); err != nil {
			return nil, err
		}
		posts = append(posts, &post)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return posts, nil
}

func (p *Pgx) GetLatestByUsername(ctx context.Context, username string, count int) ([]*domain.PostParser, error) {
	query, args, err := repositories.SqBuilder.
		Select("id", "post_id", "username", "post_url", "created_at").
		From("post_parsers").
		Where(sq.Eq{"username": username}).
		OrderBy("created_at DESC").
		Limit(uint64(count)).
		ToSql()
	if err != nil {
		return nil, repositories.ErrBadQuery
	}

	rows, err := p.querier.Query(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var posts []*domain.PostParser
	for rows.Next() {
		var post domain.PostParser
		if err := rows.Scan(&post.ID, &post.PostID, &post.Username, &post.PostURL, &post.CreatedAt); err != nil {
			return nil, err
		}
		posts = append(posts, &post)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return posts, nil
}

func (p *Pgx) Exists(ctx context.Context, postID string) (bool, error) {
	query, args, err := repositories.SqBuilder.
		Select("1").
		From("post_parsers").
		Where(sq.Eq{"post_id": postID}).
		Limit(1).
		ToSql()
	if err != nil {
		return false, repositories.ErrBadQuery
	}

	var exists bool
	err = p.querier.QueryRow(ctx, query, args...).Scan(&exists)
	if err != nil {
		if err.Error() == "no rows in result set" {
			return false, nil
		}
		return false, err
	}

	return true, nil
}

func (p *Pgx) CleanupOldRecords(ctx context.Context, olderThan string) (int64, error) {
	cutoffTime := time.Now().Add(-parseDuration(olderThan))

	query, args, err := repositories.SqBuilder.
		Delete("post_parsers").
		Where(sq.Lt{"created_at": cutoffTime}).
		ToSql()
	if err != nil {
		return 0, repositories.ErrBadQuery
	}

	result, err := p.querier.Exec(ctx, query, args...)
	if err != nil {
		return 0, err
	}

	return result.RowsAffected(), nil
}

func parseDuration(duration string) time.Duration {
	d, err := time.ParseDuration(duration)
	if err != nil {
		return 7 * 24 * time.Hour
	}
	return d
}
