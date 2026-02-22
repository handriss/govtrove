package repository

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/handriss/govtrove/api/internal/models"
)

type SavedSearchRepository struct {
	pool *pgxpool.Pool
}

func NewSavedSearchRepository(pool *pgxpool.Pool) *SavedSearchRepository {
	return &SavedSearchRepository{pool: pool}
}

func (r *SavedSearchRepository) List(ctx context.Context, userID int) ([]models.SavedSearch, error) {
	query := `SELECT id, user_id, name, filters, created_at, updated_at FROM saved_searches WHERE user_id = $1 ORDER BY updated_at DESC`

	rows, err := r.pool.Query(ctx, query, userID)
	if err != nil {
		return nil, fmt.Errorf("listing saved searches: %w", err)
	}
	defer rows.Close()

	var searches []models.SavedSearch
	for rows.Next() {
		var s models.SavedSearch
		if err := rows.Scan(&s.ID, &s.UserID, &s.Name, &s.Filters, &s.CreatedAt, &s.UpdatedAt); err != nil {
			return nil, fmt.Errorf("scanning saved search: %w", err)
		}
		searches = append(searches, s)
	}
	return searches, rows.Err()
}

func (r *SavedSearchRepository) Create(ctx context.Context, userID int, input *models.CreateSavedSearchInput) (*models.SavedSearch, error) {
	query := `
		INSERT INTO saved_searches (user_id, name, filters)
		VALUES ($1, $2, $3)
		RETURNING id, user_id, name, filters, created_at, updated_at
	`

	var s models.SavedSearch
	err := r.pool.QueryRow(ctx, query, userID, input.Name, input.Filters).Scan(
		&s.ID, &s.UserID, &s.Name, &s.Filters, &s.CreatedAt, &s.UpdatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("creating saved search: %w", err)
	}
	return &s, nil
}

func (r *SavedSearchRepository) Update(ctx context.Context, id, userID int, input *models.UpdateSavedSearchInput) (*models.SavedSearch, error) {
	query := `
		UPDATE saved_searches
		SET name = COALESCE($3, name),
		    filters = COALESCE($4, filters),
		    updated_at = NOW()
		WHERE id = $1 AND user_id = $2
		RETURNING id, user_id, name, filters, created_at, updated_at
	`

	var s models.SavedSearch
	err := r.pool.QueryRow(ctx, query, id, userID, input.Name, input.Filters).Scan(
		&s.ID, &s.UserID, &s.Name, &s.Filters, &s.CreatedAt, &s.UpdatedAt,
	)
	if err == pgx.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("updating saved search: %w", err)
	}
	return &s, nil
}

func (r *SavedSearchRepository) Delete(ctx context.Context, id, userID int) error {
	query := `DELETE FROM saved_searches WHERE id = $1 AND user_id = $2`
	_, err := r.pool.Exec(ctx, query, id, userID)
	if err != nil {
		return fmt.Errorf("deleting saved search: %w", err)
	}
	return nil
}
