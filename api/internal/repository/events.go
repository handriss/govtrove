package repository

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/opscout/api/internal/models"
)

type EventRepository struct {
	pool *pgxpool.Pool
}

func NewEventRepository(pool *pgxpool.Pool) *EventRepository {
	return &EventRepository{pool: pool}
}

func (r *EventRepository) Create(ctx context.Context, event *models.SearchEvent) error {
	filtersJSON, err := json.Marshal(event.Filters)
	if err != nil {
		return fmt.Errorf("marshaling filters: %w", err)
	}

	query := `
		INSERT INTO search_events (
			event_type, session_id, query, filters, sort_by, page,
			total_results, result_position, opportunity_id,
			user_agent, referer
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
	`

	_, err = r.pool.Exec(ctx, query,
		event.EventType,
		event.SessionID,
		event.Query,
		filtersJSON,
		event.SortBy,
		event.Page,
		event.TotalResults,
		event.ResultPosition,
		event.OpportunityID,
		event.UserAgent,
		event.Referer,
	)
	if err != nil {
		return fmt.Errorf("inserting event: %w", err)
	}

	return nil
}
