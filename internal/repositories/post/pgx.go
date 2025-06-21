package post

import (
	"context"
	"errors"
	"time"

	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/orgball2608/insta-parser-telegram-bot/internal/domain"
	"github.com/orgball2608/insta-parser-telegram-bot/internal/repositories"
	"github.com/orgball2608/insta-parser-telegram-bot/pkg/logger"

	sq "github.com/Masterminds/squirrel"
)

type Pgx struct {
	pg     *pgxpool.Pool
	logger logger.Logger
}

func NewPgx(pg *pgxpool.Pool, logger logger.Logger) *Pgx {
	return &Pgx{
		pg:     pg,
		logger: logger.WithComponent("PostParserRepo"),
	}
}

var _ Repository = (*Pgx)(nil)

// Create adds a new post parser entry
func (p *Pgx) Create(ctx context.Context, post domain.PostParser) error {
	query, args, err := repositories.SqBuilder.
		Insert("post_parsers").
		Columns("post_id", "username", "post_url", "created_at").
		Values(post.PostID, post.Username, post.PostURL, time.Now()).
		ToSql()
	if err != nil {
		return repositories.ErrBadQuery
	}

	_, err = p.pg.Exec(ctx, query, args...)
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			return ErrAlreadyExists
		}
		return err
	}
	return nil
}

// GetByUsername returns all posts for a specific username
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

	rows, err := p.pg.Query(ctx, query, args...)
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

// GetLatestByUsername returns the most recent posts for a specific username, limited by count
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

	rows, err := p.pg.Query(ctx, query, args...)
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

// Exists checks if a post with the given ID already exists
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
	err = p.pg.QueryRow(ctx, query, args...).Scan(&exists)
	if err != nil {
		if err.Error() == "no rows in result set" {
			return false, nil
		}
		return false, err
	}

	return true, nil
}

// CleanupOldRecords deletes records older than the specified duration
func (p *Pgx) CleanupOldRecords(ctx context.Context, olderThan string) (int64, error) {
	cutoffTime := time.Now().Add(-parseDuration(olderThan))

	query, args, err := repositories.SqBuilder.
		Delete("post_parsers").
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

// Helper function to parse duration strings
func parseDuration(duration string) time.Duration {
	d, err := time.ParseDuration(duration)
	if err != nil {
		// Default to 7 days if parsing fails
		return 7 * 24 * time.Hour
	}
	return d
}
