package currentstory

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

func (r *PgxRepository) GetByID(ctx context.Context, id int) (*domain.CurrentStory, error) {
	query := `
		SELECT id, username, media_url, created_at
		FROM current_stories
		WHERE id = $1
	`

	var currentStory domain.CurrentStory
	err := r.pool.QueryRow(ctx, query, id).Scan(
		&currentStory.ID,
		&currentStory.UserName,
		&currentStory.MediaURL,
		&currentStory.CreatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("failed to get current story by id: %w", err)
	}

	return &currentStory, nil
}

func (r *PgxRepository) GetByUserName(ctx context.Context, userName string) ([]*domain.CurrentStory, error) {
	query := `
		SELECT id, username, media_url, created_at
		FROM current_stories
		WHERE username = $1
		ORDER BY created_at DESC
	`

	rows, err := r.pool.Query(ctx, query, userName)
	if err != nil {
		return nil, fmt.Errorf("failed to query current stories by username: %w", err)
	}
	defer rows.Close()

	var currentStoryList []*domain.CurrentStory
	for rows.Next() {
		var currentStory domain.CurrentStory
		err := rows.Scan(
			&currentStory.ID,
			&currentStory.UserName,
			&currentStory.MediaURL,
			&currentStory.CreatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan current story row: %w", err)
		}
		currentStoryList = append(currentStoryList, &currentStory)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating current story rows: %w", err)
	}

	if len(currentStoryList) == 0 {
		return nil, ErrNotFound
	}

	return currentStoryList, nil
}

func (r *PgxRepository) Create(ctx context.Context, currentStory domain.CurrentStory) error {
	query := `
		INSERT INTO current_stories (username, media_url, created_at)
		VALUES ($1, $2, $3)
		RETURNING id
	`

	var id int
	err := r.pool.QueryRow(
		ctx,
		query,
		currentStory.UserName,
		currentStory.MediaURL,
		time.Now(),
	).Scan(&id)

	if err != nil {
		return fmt.Errorf("failed to create current story: %w", err)
	}

	return nil
}

func (r *PgxRepository) DeleteByUserName(ctx context.Context, userName string) error {
	query := `
		DELETE FROM current_stories
		WHERE username = $1
	`

	_, err := r.pool.Exec(ctx, query, userName)
	if err != nil {
		return fmt.Errorf("failed to delete current stories for user %s: %w", userName, err)
	}

	return nil
}

var _ Repository = (*PgxRepository)(nil)
