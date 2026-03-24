package repository

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/handriss/govtrove/api/internal/models"
)

type AccountRequestRepository struct {
	pool *pgxpool.Pool
}

func NewAccountRequestRepository(pool *pgxpool.Pool) *AccountRequestRepository {
	return &AccountRequestRepository{pool: pool}
}

func (r *AccountRequestRepository) Create(ctx context.Context, req *models.AccountRequest) error {
	query := `
		INSERT INTO account_requests (user_id, request_type)
		VALUES ($1, $2)
		RETURNING id, created_at
	`

	err := r.pool.QueryRow(ctx, query,
		req.UserID,
		req.RequestType,
	).Scan(&req.ID, &req.CreatedAt)
	if err != nil {
		return fmt.Errorf("inserting account request: %w", err)
	}

	return nil
}

func (r *AccountRequestRepository) ListAll(ctx context.Context, status string) ([]models.AccountRequestWithUser, error) {
	query := `
		SELECT ar.id, ar.user_id, ar.request_type, ar.status, ar.created_at,
		       ar.completed_at, ar.completed_by, ar.export_s3_key,
		       u.email, COALESCE(u.first_name || ' ' || u.last_name, '') as user_name
		FROM account_requests ar
		JOIN users u ON u.id = ar.user_id
	`
	args := []any{}
	if status != "" {
		query += " WHERE ar.status = $1"
		args = append(args, status)
	}
	query += ` ORDER BY
		CASE WHEN ar.status = 'pending' THEN 0 ELSE 1 END,
		ar.created_at DESC`

	rows, err := r.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("listing account requests: %w", err)
	}
	defer rows.Close()

	var results []models.AccountRequestWithUser
	for rows.Next() {
		var ar models.AccountRequestWithUser
		if err := rows.Scan(
			&ar.ID, &ar.UserID, &ar.RequestType, &ar.Status, &ar.CreatedAt,
			&ar.CompletedAt, &ar.CompletedBy, &ar.ExportS3Key,
			&ar.UserEmail, &ar.UserName,
		); err != nil {
			return nil, fmt.Errorf("scanning account request: %w", err)
		}
		results = append(results, ar)
	}
	return results, nil
}

func (r *AccountRequestRepository) GetByID(ctx context.Context, id int) (*models.AccountRequestWithUser, error) {
	query := `
		SELECT ar.id, ar.user_id, ar.request_type, ar.status, ar.created_at,
		       ar.completed_at, ar.completed_by, ar.export_s3_key,
		       u.email, COALESCE(u.first_name || ' ' || u.last_name, '') as user_name
		FROM account_requests ar
		JOIN users u ON u.id = ar.user_id
		WHERE ar.id = $1
	`
	var ar models.AccountRequestWithUser
	err := r.pool.QueryRow(ctx, query, id).Scan(
		&ar.ID, &ar.UserID, &ar.RequestType, &ar.Status, &ar.CreatedAt,
		&ar.CompletedAt, &ar.CompletedBy, &ar.ExportS3Key,
		&ar.UserEmail, &ar.UserName,
	)
	if err != nil {
		return nil, fmt.Errorf("getting account request: %w", err)
	}
	return &ar, nil
}

func (r *AccountRequestRepository) Complete(ctx context.Context, id int, completedBy string, exportS3Key *string) error {
	query := `
		UPDATE account_requests
		SET status = 'completed', completed_at = NOW(), completed_by = $2, export_s3_key = $3
		WHERE id = $1
	`
	_, err := r.pool.Exec(ctx, query, id, completedBy, exportS3Key)
	if err != nil {
		return fmt.Errorf("completing account request: %w", err)
	}
	return nil
}
