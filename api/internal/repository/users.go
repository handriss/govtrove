package repository

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/handriss/govtrove/api/internal/models"
)

type UserRepository struct {
	pool *pgxpool.Pool
}

func NewUserRepository(pool *pgxpool.Pool) *UserRepository {
	return &UserRepository{pool: pool}
}

func (r *UserRepository) Upsert(ctx context.Context, input *models.UpsertUserInput) (*models.User, error) {
	query := `
		INSERT INTO users (workos_id, email, first_name, last_name)
		VALUES ($1, $2, $3, $4)
		ON CONFLICT (workos_id) DO UPDATE SET
			email = EXCLUDED.email,
			first_name = EXCLUDED.first_name,
			last_name = EXCLUDED.last_name,
			updated_at = NOW()
		RETURNING id, workos_id, email, first_name, last_name, plan, created_at, updated_at
	`

	var u models.User
	err := r.pool.QueryRow(ctx, query,
		input.WorkOSID,
		input.Email,
		input.FirstName,
		input.LastName,
	).Scan(&u.ID, &u.WorkOSID, &u.Email, &u.FirstName, &u.LastName, &u.Plan, &u.CreatedAt, &u.UpdatedAt)
	if err != nil {
		return nil, fmt.Errorf("upserting user: %w", err)
	}
	return &u, nil
}

func (r *UserRepository) GetByWorkOSID(ctx context.Context, workosID string) (*models.User, error) {
	query := `
		SELECT id, workos_id, email, first_name, last_name, plan, created_at, updated_at
		FROM users WHERE workos_id = $1
	`

	var u models.User
	err := r.pool.QueryRow(ctx, query, workosID).Scan(
		&u.ID, &u.WorkOSID, &u.Email, &u.FirstName, &u.LastName, &u.Plan, &u.CreatedAt, &u.UpdatedAt,
	)
	if err == pgx.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("getting user by workos_id: %w", err)
	}
	return &u, nil
}

func (r *UserRepository) GetByID(ctx context.Context, id int) (*models.User, error) {
	query := `
		SELECT id, workos_id, email, first_name, last_name, plan, created_at, updated_at
		FROM users WHERE id = $1
	`

	var u models.User
	err := r.pool.QueryRow(ctx, query, id).Scan(
		&u.ID, &u.WorkOSID, &u.Email, &u.FirstName, &u.LastName, &u.Plan, &u.CreatedAt, &u.UpdatedAt,
	)
	if err == pgx.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("getting user by id: %w", err)
	}
	return &u, nil
}
