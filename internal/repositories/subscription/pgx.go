package subscription

import (
	"context"
	"errors"
	"strings"

	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/orgball2608/insta-parser-telegram-bot/internal/domain"
	"github.com/orgball2608/insta-parser-telegram-bot/internal/repositories"
	"github.com/orgball2608/insta-parser-telegram-bot/pkg/logger"

	sq "github.com/Masterminds/squirrel"
)

type PgxRepository struct {
	pool   *pgxpool.Pool
	logger logger.Logger
}

func NewPgxRepository(pool *pgxpool.Pool, logger logger.Logger) *PgxRepository {
	return &PgxRepository{
		pool:   pool,
		logger: logger.WithComponent("SubscriptionRepo"),
	}
}

var _ Repository = (*PgxRepository)(nil)

func (r *PgxRepository) Create(ctx context.Context, sub domain.Subscription) error {
	// Set default subscription type if not specified
	if sub.SubscriptionType == "" {
		sub.SubscriptionType = domain.SubscriptionTypeStory
	}

	query, args, err := repositories.SqBuilder.
		Insert("subscriptions").
		Columns("chat_id", "instagram_username", "subscription_type").
		Values(sub.ChatID, sub.InstagramUsername, sub.SubscriptionType).
		ToSql()
	if err != nil {
		return repositories.ErrBadQuery
	}

	_, err = r.pool.Exec(ctx, query, args...)
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			return ErrAlreadyExists
		}
		return err
	}
	return nil
}

func (r *PgxRepository) Delete(ctx context.Context, chatID int64, username string) error {
	query, args, err := repositories.SqBuilder.
		Delete("subscriptions").
		Where(sq.Eq{"chat_id": chatID, "instagram_username": username}).
		ToSql()
	if err != nil {
		return repositories.ErrBadQuery
	}

	result, err := r.pool.Exec(ctx, query, args...)
	if err != nil {
		return err
	}

	if result.RowsAffected() == 0 {
		return ErrNotFound
	}

	return nil
}

func (r *PgxRepository) GetByChatID(ctx context.Context, chatID int64) ([]*domain.Subscription, error) {
	query, args, err := repositories.SqBuilder.
		Select("id", "chat_id", "instagram_username", "subscription_type", "created_at").
		From("subscriptions").
		Where(sq.Eq{"chat_id": chatID}).
		OrderBy("instagram_username ASC").
		ToSql()
	if err != nil {
		return nil, repositories.ErrBadQuery
	}

	rows, err := r.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var subs []*domain.Subscription
	for rows.Next() {
		var sub domain.Subscription
		if err := rows.Scan(&sub.ID, &sub.ChatID, &sub.InstagramUsername, &sub.SubscriptionType, &sub.CreatedAt); err != nil {
			return nil, err
		}
		subs = append(subs, &sub)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return subs, nil
}

func (r *PgxRepository) GetAllUniqueUsernames(ctx context.Context) ([]string, error) {
	query := `SELECT DISTINCT instagram_username FROM subscriptions`

	rows, err := r.pool.Query(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var usernames []string
	for rows.Next() {
		var username string
		if err := rows.Scan(&username); err != nil {
			return nil, err
		}
		usernames = append(usernames, username)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return usernames, nil
}

func (r *PgxRepository) GetSubscribersForUser(ctx context.Context, username string) ([]int64, error) {
	query, args, err := repositories.SqBuilder.
		Select("chat_id").
		From("subscriptions").
		Where(sq.Eq{"instagram_username": username}).
		ToSql()
	if err != nil {
		return nil, repositories.ErrBadQuery
	}

	rows, err := r.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var chatIDs []int64
	for rows.Next() {
		var chatID int64
		if err := rows.Scan(&chatID); err != nil {
			return nil, err
		}
		chatIDs = append(chatIDs, chatID)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return chatIDs, nil
}

// GetSubscribersForUserByType returns chat IDs of users subscribed to a specific username with a specific subscription type
func (r *PgxRepository) GetSubscribersForUserByType(ctx context.Context, username string, subscriptionType string) ([]int64, error) {
	var builder sq.SelectBuilder
	if subscriptionType == domain.SubscriptionTypeAll {
		builder = repositories.SqBuilder.
			Select("chat_id").
			From("subscriptions").
			Where(sq.Eq{"instagram_username": username}).
			Where(sq.Or{
				sq.Eq{"subscription_type": domain.SubscriptionTypeAll},
				sq.Eq{"subscription_type": subscriptionType},
			})
	} else {
		builder = repositories.SqBuilder.
			Select("chat_id").
			From("subscriptions").
			Where(sq.Eq{"instagram_username": username}).
			Where(sq.Or{
				sq.Eq{"subscription_type": domain.SubscriptionTypeAll},
				sq.Eq{"subscription_type": subscriptionType},
			})
	}

	query, args, err := builder.ToSql()
	if err != nil {
		return nil, repositories.ErrBadQuery
	}

	rows, err := r.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var chatIDs []int64
	for rows.Next() {
		var chatID int64
		if err := rows.Scan(&chatID); err != nil {
			return nil, err
		}
		chatIDs = append(chatIDs, chatID)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return chatIDs, nil
}

// GetAllUniqueUsernamesByType returns all unique usernames with a specific subscription type
func (r *PgxRepository) GetAllUniqueUsernamesByType(ctx context.Context, subscriptionType string) ([]string, error) {
	var query string
	var args []interface{}
	var err error

	if subscriptionType == "" {
		query = `SELECT DISTINCT instagram_username FROM subscriptions`
	} else {
		builder := repositories.SqBuilder.
			Select("DISTINCT instagram_username").
			From("subscriptions").
			Where(sq.Or{
				sq.Eq{"subscription_type": domain.SubscriptionTypeAll},
				sq.Eq{"subscription_type": subscriptionType},
			})

		query, args, err = builder.ToSql()
		if err != nil {
			return nil, repositories.ErrBadQuery
		}
	}

	rows, err := r.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var usernames []string
	for rows.Next() {
		var username string
		if err := rows.Scan(&username); err != nil {
			return nil, err
		}
		usernames = append(usernames, username)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return usernames, nil
}

// UpdateSubscriptionType updates the subscription type for a specific chat ID and username
func (r *PgxRepository) UpdateSubscriptionType(ctx context.Context, chatID int64, username string, subscriptionType string) error {
	query, args, err := repositories.SqBuilder.
		Update("subscriptions").
		Set("subscription_type", subscriptionType).
		Where(sq.Eq{
			"chat_id":            chatID,
			"instagram_username": username,
		}).
		ToSql()
	if err != nil {
		return repositories.ErrBadQuery
	}

	result, err := r.pool.Exec(ctx, query, args...)
	if err != nil {
		return err
	}

	if result.RowsAffected() == 0 {
		return ErrNotFound
	}

	return nil
}

// Helper to sanitize username input
func SanitizeUsername(username string) string {
	return strings.ToLower(strings.Trim(username, "@ "))
}
