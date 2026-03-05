package repository

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/handriss/govtrove/api/internal/models"
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
			event_type, user_id, query, filters, sort_by, page,
			total_results, opportunity_id, duration_ms,
			user_agent, referer, utm_campaign
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12)
	`

	_, err = r.pool.Exec(ctx, query,
		event.EventType,
		event.UserID,
		event.Query,
		filtersJSON,
		event.SortBy,
		event.Page,
		event.TotalResults,
		event.OpportunityID,
		event.DurationMs,
		event.UserAgent,
		event.Referer,
		event.UtmCampaign,
	)
	if err != nil {
		return fmt.Errorf("inserting event: %w", err)
	}

	return nil
}

func (r *EventRepository) DeleteOlderThan(ctx context.Context, days int) (int64, error) {
	tag, err := r.pool.Exec(ctx,
		`DELETE FROM search_events WHERE created_at < NOW() - INTERVAL '1 day' * $1`,
		days,
	)
	if err != nil {
		return 0, fmt.Errorf("deleting old events: %w", err)
	}
	return tag.RowsAffected(), nil
}
