package repository

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"
)

type Agency struct {
	ID               int      `json:"id"`
	Name             string   `json:"name"`
	ShortName        *string  `json:"short_name,omitempty"`
	Level            string   `json:"level"`
	ParentPath       string   `json:"parent_path"`
	Aliases          []string `json:"-"`
	OpportunityCount int      `json:"count"`
	Breadcrumb       string   `json:"breadcrumb"`
}

type AgencyRepository struct {
	pool *pgxpool.Pool
}

func NewAgencyRepository(pool *pgxpool.Pool) *AgencyRepository {
	return &AgencyRepository{pool: pool}
}

func (r *AgencyRepository) Search(ctx context.Context, q string, limit int) ([]Agency, error) {
	if limit <= 0 || limit > 50 {
		limit = 15
	}

	var query string
	var args []any

	if q == "" {
		query = `
			SELECT id, name, short_name, level, parent_path, aliases, opportunity_count
			FROM agencies
			WHERE opportunity_count > 0
			ORDER BY opportunity_count DESC
			LIMIT $1
		`
		args = []any{limit}
	} else {
		query = `
			SELECT id, name, short_name, level, parent_path, aliases, opportunity_count
			FROM agencies
			WHERE opportunity_count > 0 AND (
				name ILIKE '%' || $1 || '%'
				OR short_name ILIKE '%' || $1 || '%'
				OR EXISTS (SELECT 1 FROM unnest(aliases) a WHERE a ILIKE '%' || $1 || '%')
			)
			ORDER BY
				CASE
					WHEN short_name ILIKE $1 THEN 0
					WHEN name ILIKE $1 || '%' OR short_name ILIKE $1 || '%' THEN 1
					WHEN name ILIKE '%' || $1 || '%' THEN 2
					ELSE 3
				END,
				opportunity_count DESC
			LIMIT $2
		`
		args = []any{q, limit}
	}

	rows, err := r.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("search agencies: %w", err)
	}
	defer rows.Close()

	var agencies []Agency
	for rows.Next() {
		var a Agency
		if err := rows.Scan(&a.ID, &a.Name, &a.ShortName, &a.Level, &a.ParentPath, &a.Aliases, &a.OpportunityCount); err != nil {
			return nil, fmt.Errorf("scan agency: %w", err)
		}
		agencies = append(agencies, a)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate agencies: %w", err)
	}

	return agencies, nil
}
