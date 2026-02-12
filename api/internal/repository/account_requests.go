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
