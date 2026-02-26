package repository

import (
	"context"
	"fmt"
	"time"

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

type UpsertResult struct {
	User  *models.User
	IsNew bool
}

func (r *UserRepository) Upsert(ctx context.Context, input *models.UpsertUserInput) (*UpsertResult, error) {
	// xmax = 0 means the row was inserted (not updated)
	query := `
		INSERT INTO users (workos_id, email, first_name, last_name)
		VALUES ($1, $2, $3, $4)
		ON CONFLICT (workos_id) DO UPDATE SET
			email = EXCLUDED.email,
			first_name = EXCLUDED.first_name,
			last_name = EXCLUDED.last_name,
			updated_at = NOW()
		RETURNING id, workos_id, email, first_name, last_name, plan, created_at, updated_at, (xmax = 0) AS is_new
	`

	var u models.User
	var isNew bool
	err := r.pool.QueryRow(ctx, query,
		input.WorkOSID,
		input.Email,
		input.FirstName,
		input.LastName,
	).Scan(&u.ID, &u.WorkOSID, &u.Email, &u.FirstName, &u.LastName, &u.Plan, &u.CreatedAt, &u.UpdatedAt, &isNew)
	if err != nil {
		return nil, fmt.Errorf("upserting user: %w", err)
	}
	return &UpsertResult{User: &u, IsNew: isNew}, nil
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

func (r *UserRepository) IsAdmin(ctx context.Context, workosID string) (bool, error) {
	var isAdmin bool
	err := r.pool.QueryRow(ctx, `SELECT is_admin FROM users WHERE workos_id = $1`, workosID).Scan(&isAdmin)
	if err == pgx.ErrNoRows {
		return false, nil
	}
	if err != nil {
		return false, fmt.Errorf("checking admin status: %w", err)
	}
	return isAdmin, nil
}

type AdminUserRow struct {
	ID        int
	Email     string
	FirstName string
	LastName  string
	Plan      string
	IsAdmin   bool
	CreatedAt time.Time
	UpdatedAt time.Time
}

func (r *UserRepository) ListUsers(ctx context.Context) ([]AdminUserRow, error) {
	query := `
		SELECT id, email, first_name, last_name, plan, is_admin, created_at, updated_at
		FROM users ORDER BY created_at DESC
	`

	rows, err := r.pool.Query(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("listing users: %w", err)
	}
	defer rows.Close()

	var users []AdminUserRow
	for rows.Next() {
		var u AdminUserRow
		if err := rows.Scan(&u.ID, &u.Email, &u.FirstName, &u.LastName, &u.Plan, &u.IsAdmin, &u.CreatedAt, &u.UpdatedAt); err != nil {
			return nil, fmt.Errorf("scanning user: %w", err)
		}
		users = append(users, u)
	}
	return users, rows.Err()
}
