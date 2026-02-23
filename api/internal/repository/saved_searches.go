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
	query := `SELECT id, user_id, name, filters, alert_enabled, last_checked_at, last_match_count, total_result_count, created_at, updated_at
		FROM saved_searches WHERE user_id = $1 ORDER BY updated_at DESC`

	rows, err := r.pool.Query(ctx, query, userID)
	if err != nil {
		return nil, fmt.Errorf("listing saved searches: %w", err)
	}
	defer rows.Close()

	var searches []models.SavedSearch
	for rows.Next() {
		var s models.SavedSearch
		if err := rows.Scan(&s.ID, &s.UserID, &s.Name, &s.Filters,
			&s.AlertEnabled, &s.LastCheckedAt, &s.LastMatchCount, &s.TotalResultCount,
			&s.CreatedAt, &s.UpdatedAt); err != nil {
			return nil, fmt.Errorf("scanning saved search: %w", err)
		}
		searches = append(searches, s)
	}
	return searches, rows.Err()
}

func (r *SavedSearchRepository) Create(ctx context.Context, userID int, input *models.CreateSavedSearchInput) (*models.SavedSearch, error) {
	alertEnabled := true
	if input.AlertEnabled != nil {
		alertEnabled = *input.AlertEnabled
	}

	query := `
		INSERT INTO saved_searches (user_id, name, filters, alert_enabled)
		VALUES ($1, $2, $3, $4)
		RETURNING id, user_id, name, filters, alert_enabled, last_checked_at, last_match_count, total_result_count, created_at, updated_at
	`

	var s models.SavedSearch
	err := r.pool.QueryRow(ctx, query, userID, input.Name, input.Filters, alertEnabled).Scan(
		&s.ID, &s.UserID, &s.Name, &s.Filters,
		&s.AlertEnabled, &s.LastCheckedAt, &s.LastMatchCount, &s.TotalResultCount,
		&s.CreatedAt, &s.UpdatedAt,
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
		    alert_enabled = COALESCE($5, alert_enabled),
		    updated_at = NOW()
		WHERE id = $1 AND user_id = $2
		RETURNING id, user_id, name, filters, alert_enabled, last_checked_at, last_match_count, total_result_count, created_at, updated_at
	`

	var s models.SavedSearch
	err := r.pool.QueryRow(ctx, query, id, userID, input.Name, input.Filters, input.AlertEnabled).Scan(
		&s.ID, &s.UserID, &s.Name, &s.Filters,
		&s.AlertEnabled, &s.LastCheckedAt, &s.LastMatchCount, &s.TotalResultCount,
		&s.CreatedAt, &s.UpdatedAt,
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
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback(ctx)

	_, err = tx.Exec(ctx, `DELETE FROM user_updates WHERE source_id = $1 AND update_type = 'saved_search_matches'`, id)
	if err != nil {
		return fmt.Errorf("deleting related updates: %w", err)
	}

	_, err = tx.Exec(ctx, `DELETE FROM saved_searches WHERE id = $1 AND user_id = $2`, id, userID)
	if err != nil {
		return fmt.Errorf("deleting saved search: %w", err)
	}

	return tx.Commit(ctx)
}

func (r *SavedSearchRepository) Run(ctx context.Context, id, userID int) (*models.SavedSearch, *models.SearchResult, error) {
	var s models.SavedSearch
	err := r.pool.QueryRow(ctx,
		`SELECT id, user_id, name, filters, alert_enabled, last_checked_at, last_match_count, total_result_count, created_at, updated_at
		 FROM saved_searches WHERE id = $1 AND user_id = $2`, id, userID,
	).Scan(&s.ID, &s.UserID, &s.Name, &s.Filters,
		&s.AlertEnabled, &s.LastCheckedAt, &s.LastMatchCount, &s.TotalResultCount,
		&s.CreatedAt, &s.UpdatedAt)
	if err == pgx.ErrNoRows {
		return nil, nil, nil
	}
	if err != nil {
		return nil, nil, fmt.Errorf("fetching saved search: %w", err)
	}

	params, err := ParseSavedFilters(s.Filters)
	if err != nil {
		return nil, nil, fmt.Errorf("parsing filters: %w", err)
	}
	params.Page = 1
	params.Limit = 25

	oppRepo := NewOpportunityRepository(r.pool)
	result, err := oppRepo.Search(ctx, params)
	if err != nil {
		return nil, nil, fmt.Errorf("running search: %w", err)
	}

	totalCount := result.Total
	_, err = r.pool.Exec(ctx,
		`UPDATE saved_searches SET total_result_count = $1, updated_at = NOW() WHERE id = $2`,
		totalCount, s.ID)
	if err != nil {
		return nil, nil, fmt.Errorf("updating total_result_count: %w", err)
	}
	s.TotalResultCount = &totalCount

	return &s, result, nil
}
