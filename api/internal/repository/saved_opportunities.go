package repository

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"
)

type SavedOpportunityRepository struct {
	pool *pgxpool.Pool
}

func NewSavedOpportunityRepository(pool *pgxpool.Pool) *SavedOpportunityRepository {
	return &SavedOpportunityRepository{pool: pool}
}

func (r *SavedOpportunityRepository) List(ctx context.Context, userID int) ([]int, error) {
	query := `SELECT opportunity_id FROM saved_opportunities WHERE user_id = $1 ORDER BY created_at DESC`

	rows, err := r.pool.Query(ctx, query, userID)
	if err != nil {
		return nil, fmt.Errorf("listing saved opportunities: %w", err)
	}
	defer rows.Close()

	var ids []int
	for rows.Next() {
		var id int
		if err := rows.Scan(&id); err != nil {
			return nil, fmt.Errorf("scanning saved opportunity: %w", err)
		}
		ids = append(ids, id)
	}
	return ids, rows.Err()
}

func (r *SavedOpportunityRepository) Add(ctx context.Context, userID, opportunityID int) error {
	query := `INSERT INTO saved_opportunities (user_id, opportunity_id) VALUES ($1, $2) ON CONFLICT DO NOTHING`
	_, err := r.pool.Exec(ctx, query, userID, opportunityID)
	if err != nil {
		return fmt.Errorf("adding saved opportunity: %w", err)
	}
	return nil
}

func (r *SavedOpportunityRepository) Remove(ctx context.Context, userID, opportunityID int) error {
	query := `DELETE FROM saved_opportunities WHERE user_id = $1 AND opportunity_id = $2`
	_, err := r.pool.Exec(ctx, query, userID, opportunityID)
	if err != nil {
		return fmt.Errorf("removing saved opportunity: %w", err)
	}
	return nil
}

func (r *SavedOpportunityRepository) BulkAdd(ctx context.Context, userID int, opportunityIDs []int) error {
	if len(opportunityIDs) == 0 {
		return nil
	}

	query := `INSERT INTO saved_opportunities (user_id, opportunity_id) SELECT $1, unnest($2::int[]) ON CONFLICT DO NOTHING`
	_, err := r.pool.Exec(ctx, query, userID, opportunityIDs)
	if err != nil {
		return fmt.Errorf("bulk adding saved opportunities: %w", err)
	}
	return nil
}
