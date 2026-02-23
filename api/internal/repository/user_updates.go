package repository

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/handriss/govtrove/api/internal/models"
)

type UserUpdateRepository struct {
	pool *pgxpool.Pool
}

func NewUserUpdateRepository(pool *pgxpool.Pool) *UserUpdateRepository {
	return &UserUpdateRepository{pool: pool}
}

func (r *UserUpdateRepository) List(ctx context.Context, userID int, unreadOnly bool, page, limit int) ([]models.UserUpdate, int, error) {
	if page < 1 {
		page = 1
	}
	if limit < 1 || limit > 100 {
		limit = 25
	}

	conditions := "user_id = $1"
	args := []any{userID}
	if unreadOnly {
		conditions += " AND is_read = false"
	}

	var total int
	countQuery := fmt.Sprintf("SELECT COUNT(*) FROM user_updates WHERE %s", conditions)
	if err := r.pool.QueryRow(ctx, countQuery, args...).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("counting user updates: %w", err)
	}

	query := fmt.Sprintf(`
		SELECT id, user_id, update_type, source_id, opportunity_ids, summary, details, is_read, created_at
		FROM user_updates
		WHERE %s
		ORDER BY created_at DESC
		LIMIT $2 OFFSET $3
	`, conditions)
	args = append(args, limit, (page-1)*limit)

	rows, err := r.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("listing user updates: %w", err)
	}
	defer rows.Close()

	var updates []models.UserUpdate
	for rows.Next() {
		var u models.UserUpdate
		if err := rows.Scan(&u.ID, &u.UserID, &u.UpdateType, &u.SourceID,
			&u.OpportunityIDs, &u.Summary, &u.Details, &u.IsRead, &u.CreatedAt); err != nil {
			return nil, 0, fmt.Errorf("scanning user update: %w", err)
		}
		updates = append(updates, u)
	}
	return updates, total, rows.Err()
}

func (r *UserUpdateRepository) Count(ctx context.Context, userID int) (*models.UserUpdateCount, error) {
	var c models.UserUpdateCount
	err := r.pool.QueryRow(ctx,
		`SELECT
			COUNT(*) FILTER (WHERE is_read = false),
			COUNT(*)
		FROM user_updates WHERE user_id = $1`, userID,
	).Scan(&c.Unread, &c.Total)
	if err != nil {
		return nil, fmt.Errorf("counting user updates: %w", err)
	}
	return &c, nil
}

func (r *UserUpdateRepository) MarkRead(ctx context.Context, id string, userID int) error {
	tag, err := r.pool.Exec(ctx,
		`UPDATE user_updates SET is_read = true WHERE id = $1 AND user_id = $2`,
		id, userID)
	if err != nil {
		return fmt.Errorf("marking update read: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return pgx.ErrNoRows
	}
	return nil
}

func (r *UserUpdateRepository) MarkAllRead(ctx context.Context, userID int) (int, error) {
	tag, err := r.pool.Exec(ctx,
		`UPDATE user_updates SET is_read = true WHERE user_id = $1 AND is_read = false`,
		userID)
	if err != nil {
		return 0, fmt.Errorf("marking all updates read: %w", err)
	}
	return int(tag.RowsAffected()), nil
}

func (r *UserUpdateRepository) Delete(ctx context.Context, id string, userID int) error {
	tag, err := r.pool.Exec(ctx,
		`DELETE FROM user_updates WHERE id = $1 AND user_id = $2`,
		id, userID)
	if err != nil {
		return fmt.Errorf("deleting user update: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return pgx.ErrNoRows
	}
	return nil
}
