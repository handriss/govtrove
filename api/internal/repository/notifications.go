package repository

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/handriss/govtrove/api/internal/models"
)

type NotificationRepository struct {
	pool *pgxpool.Pool
}

func NewNotificationRepository(pool *pgxpool.Pool) *NotificationRepository {
	return &NotificationRepository{pool: pool}
}

func (r *NotificationRepository) List(ctx context.Context, userID int, unreadOnly bool, page, limit int) ([]models.Notification, int, error) {
	if page < 1 {
		page = 1
	}
	if limit < 1 || limit > 100 {
		limit = 25
	}

	conditions := "user_id = $1 AND expires_at > NOW()"
	args := []any{userID}
	if unreadOnly {
		conditions += " AND is_read = false"
	}

	var total int
	countQuery := fmt.Sprintf("SELECT COUNT(*) FROM notifications WHERE %s", conditions)
	if err := r.pool.QueryRow(ctx, countQuery, args...).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("counting notifications: %w", err)
	}

	query := fmt.Sprintf(`
		SELECT id, user_id, update_type, source_id, group_key, details, is_read, expires_at, created_at
		FROM notifications
		WHERE %s
		ORDER BY created_at DESC
		LIMIT $2 OFFSET $3
	`, conditions)
	args = append(args, limit, (page-1)*limit)

	rows, err := r.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("listing notifications: %w", err)
	}
	defer rows.Close()

	var notifications []models.Notification
	for rows.Next() {
		var n models.Notification
		if err := rows.Scan(&n.ID, &n.UserID, &n.UpdateType, &n.SourceID,
			&n.GroupKey, &n.Details, &n.IsRead, &n.ExpiresAt, &n.CreatedAt); err != nil {
			return nil, 0, fmt.Errorf("scanning notification: %w", err)
		}
		notifications = append(notifications, n)
	}
	return notifications, total, rows.Err()
}

func (r *NotificationRepository) Count(ctx context.Context, userID int) (*models.NotificationCount, error) {
	var c models.NotificationCount
	err := r.pool.QueryRow(ctx,
		`SELECT
			COUNT(*) FILTER (WHERE is_read = false),
			COUNT(*)
		FROM notifications WHERE user_id = $1 AND expires_at > NOW()`, userID,
	).Scan(&c.Unread, &c.Total)
	if err != nil {
		return nil, fmt.Errorf("counting notifications: %w", err)
	}
	return &c, nil
}

func (r *NotificationRepository) MarkRead(ctx context.Context, id string, userID int) error {
	tag, err := r.pool.Exec(ctx,
		`UPDATE notifications SET is_read = true WHERE id = $1 AND user_id = $2`,
		id, userID)
	if err != nil {
		return fmt.Errorf("marking notification read: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return pgx.ErrNoRows
	}
	return nil
}

func (r *NotificationRepository) MarkAllRead(ctx context.Context, userID int) (int, error) {
	tag, err := r.pool.Exec(ctx,
		`UPDATE notifications SET is_read = true WHERE user_id = $1 AND is_read = false`,
		userID)
	if err != nil {
		return 0, fmt.Errorf("marking all notifications read: %w", err)
	}
	return int(tag.RowsAffected()), nil
}

func (r *NotificationRepository) Delete(ctx context.Context, id string, userID int) error {
	tag, err := r.pool.Exec(ctx,
		`DELETE FROM notifications WHERE id = $1 AND user_id = $2`,
		id, userID)
	if err != nil {
		return fmt.Errorf("deleting notification: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return pgx.ErrNoRows
	}
	return nil
}

func (r *NotificationRepository) Upsert(ctx context.Context, n models.Notification) error {
	_, err := r.pool.Exec(ctx, `
		INSERT INTO notifications (id, user_id, update_type, source_id, group_key, details, is_read, expires_at, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
		ON CONFLICT (group_key) WHERE group_key IS NOT NULL DO UPDATE SET
			details = EXCLUDED.details,
			is_read = false,
			expires_at = EXCLUDED.expires_at,
			created_at = EXCLUDED.created_at
	`, n.ID, n.UserID, n.UpdateType, n.SourceID, n.GroupKey, n.Details, n.IsRead, n.ExpiresAt, n.CreatedAt)
	if err != nil {
		return fmt.Errorf("upserting notification: %w", err)
	}
	return nil
}

func (r *NotificationRepository) PurgeExpired(ctx context.Context) (int, error) {
	tag, err := r.pool.Exec(ctx, `DELETE FROM notifications WHERE expires_at < NOW()`)
	if err != nil {
		return 0, fmt.Errorf("purging expired notifications: %w", err)
	}
	return int(tag.RowsAffected()), nil
}

