package highlights

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/orgball2608/insta-parser-telegram-bot/internal/domain"
	"github.com/orgball2608/insta-parser-telegram-bot/pkg/logger"
)

type PgxRepository struct {
	pool   *pgxpool.Pool
	logger logger.Logger
}

func NewPgxRepository(pool *pgxpool.Pool, logger logger.Logger) *PgxRepository {
	return &PgxRepository{
		pool:   pool,
		logger: logger,
	}
}

func (r *PgxRepository) GetByID(ctx context.Context, id int) (*domain.Highlights, error) {
	query := `
		SELECT id, username, media_url, created_at
		FROM highlights
		WHERE id = $1
	`

	var highlights domain.Highlights
	err := r.pool.QueryRow(ctx, query, id).Scan(
		&highlights.ID,
		&highlights.UserName,
		&highlights.MediaURL,
		&highlights.CreatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("failed to get highlights by id: %w", err)
	}

	return &highlights, nil
}

func (r *PgxRepository) GetByUserName(ctx context.Context, userName string) ([]*domain.Highlights, error) {
	query := `
		SELECT id, username, media_url, created_at
		FROM highlights
		WHERE username = $1
		ORDER BY created_at DESC
	`

	rows, err := r.pool.Query(ctx, query, userName)
	if err != nil {
		return nil, fmt.Errorf("failed to query highlights by username: %w", err)
	}
	defer rows.Close()

	var highlightsList []*domain.Highlights
	for rows.Next() {
		var highlights domain.Highlights
		err := rows.Scan(
			&highlights.ID,
			&highlights.UserName,
			&highlights.MediaURL,
			&highlights.CreatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan highlights row: %w", err)
		}
		highlightsList = append(highlightsList, &highlights)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating highlights rows: %w", err)
	}

	if len(highlightsList) == 0 {
		return nil, ErrNotFound
	}

	return highlightsList, nil
}

func (r *PgxRepository) Create(ctx context.Context, highlights domain.Highlights) error {
	query := `
		INSERT INTO highlights (username, media_url, created_at)
		VALUES ($1, $2, $3)
		RETURNING id
	`

	var id int
	err := r.pool.QueryRow(
		ctx,
		query,
		highlights.UserName,
		highlights.MediaURL,
		time.Now(),
	).Scan(&id)

	if err != nil {
		return fmt.Errorf("failed to create highlights: %w", err)
	}

	return nil
}

var _ Repository = (*PgxRepository)(nil)
