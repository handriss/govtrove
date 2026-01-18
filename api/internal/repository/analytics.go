package repository

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/opscout/api/internal/models"
)

type AnalyticsRepository struct {
	pool *pgxpool.Pool
}

func NewAnalyticsRepository(pool *pgxpool.Pool) *AnalyticsRepository {
	return &AnalyticsRepository{pool: pool}
}

func (r *AnalyticsRepository) GetSearchAnalytics(ctx context.Context, period string) (*models.SearchAnalytics, error) {
	interval := periodToInterval(period)

	popularSearches, err := r.getPopularSearches(ctx, interval)
	if err != nil {
		return nil, fmt.Errorf("getting popular searches: %w", err)
	}

	zeroResultTerms, err := r.getZeroResultSearches(ctx, interval)
	if err != nil {
		return nil, fmt.Errorf("getting zero result searches: %w", err)
	}

	filterUsage, err := r.getFilterUsage(ctx, interval)
	if err != nil {
		return nil, fmt.Errorf("getting filter usage: %w", err)
	}

	clickStats, err := r.getClickStats(ctx, interval)
	if err != nil {
		return nil, fmt.Errorf("getting click stats: %w", err)
	}

	eventCounts, err := r.getEventCounts(ctx, interval)
	if err != nil {
		return nil, fmt.Errorf("getting event counts: %w", err)
	}

	return &models.SearchAnalytics{
		PopularSearches: popularSearches,
		ZeroResultTerms: zeroResultTerms,
		FilterUsage:     *filterUsage,
		ClickStats:      *clickStats,
		EventCounts:     *eventCounts,
	}, nil
}

func periodToInterval(period string) string {
	switch period {
	case "24h":
		return "24 hours"
	case "7d":
		return "7 days"
	case "30d":
		return "30 days"
	default:
		return "7 days"
	}
}

func (r *AnalyticsRepository) getPopularSearches(ctx context.Context, interval string) ([]models.SearchTermStat, error) {
	query := fmt.Sprintf(`
		SELECT LOWER(TRIM(query)) as query, COUNT(*) as count
		FROM search_events
		WHERE event_type = 'search'
		  AND query IS NOT NULL
		  AND query != ''
		  AND created_at > NOW() - INTERVAL '%s'
		GROUP BY LOWER(TRIM(query))
		ORDER BY count DESC
		LIMIT 20
	`, interval)

	rows, err := r.pool.Query(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var results []models.SearchTermStat
	for rows.Next() {
		var stat models.SearchTermStat
		if err := rows.Scan(&stat.Query, &stat.Count); err != nil {
			return nil, err
		}
		results = append(results, stat)
	}

	if results == nil {
		results = []models.SearchTermStat{}
	}
	return results, rows.Err()
}

func (r *AnalyticsRepository) getZeroResultSearches(ctx context.Context, interval string) ([]models.SearchTermStat, error) {
	query := fmt.Sprintf(`
		SELECT LOWER(TRIM(query)) as query, COUNT(*) as count
		FROM search_events
		WHERE event_type = 'search'
		  AND query IS NOT NULL
		  AND query != ''
		  AND total_results = 0
		  AND created_at > NOW() - INTERVAL '%s'
		GROUP BY LOWER(TRIM(query))
		ORDER BY count DESC
		LIMIT 20
	`, interval)

	rows, err := r.pool.Query(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var results []models.SearchTermStat
	for rows.Next() {
		var stat models.SearchTermStat
		if err := rows.Scan(&stat.Query, &stat.Count); err != nil {
			return nil, err
		}
		results = append(results, stat)
	}

	if results == nil {
		results = []models.SearchTermStat{}
	}
	return results, rows.Err()
}

func (r *AnalyticsRepository) getFilterUsage(ctx context.Context, interval string) (*models.FilterUsageStats, error) {
	stats := &models.FilterUsageStats{
		Types:     []models.FilterStat{},
		SetAsides: []models.FilterStat{},
		States:    []models.FilterStat{},
	}

	// Get type filter usage
	typeQuery := fmt.Sprintf(`
		SELECT value, COUNT(*) as count
		FROM search_events,
		     jsonb_array_elements_text(filters->'type') as value
		WHERE event_type IN ('search', 'filter')
		  AND filters->'type' IS NOT NULL
		  AND created_at > NOW() - INTERVAL '%s'
		GROUP BY value
		ORDER BY count DESC
		LIMIT 10
	`, interval)

	rows, err := r.pool.Query(ctx, typeQuery)
	if err != nil {
		return nil, err
	}
	for rows.Next() {
		var stat models.FilterStat
		if err := rows.Scan(&stat.Value, &stat.Count); err != nil {
			rows.Close()
			return nil, err
		}
		stats.Types = append(stats.Types, stat)
	}
	rows.Close()

	// Get set_aside filter usage
	setAsideQuery := fmt.Sprintf(`
		SELECT value, COUNT(*) as count
		FROM search_events,
		     jsonb_array_elements_text(filters->'set_aside') as value
		WHERE event_type IN ('search', 'filter')
		  AND filters->'set_aside' IS NOT NULL
		  AND created_at > NOW() - INTERVAL '%s'
		GROUP BY value
		ORDER BY count DESC
		LIMIT 10
	`, interval)

	rows, err = r.pool.Query(ctx, setAsideQuery)
	if err != nil {
		return nil, err
	}
	for rows.Next() {
		var stat models.FilterStat
		if err := rows.Scan(&stat.Value, &stat.Count); err != nil {
			rows.Close()
			return nil, err
		}
		stats.SetAsides = append(stats.SetAsides, stat)
	}
	rows.Close()

	// Get state filter usage
	stateQuery := fmt.Sprintf(`
		SELECT value, COUNT(*) as count
		FROM search_events,
		     jsonb_array_elements_text(filters->'state') as value
		WHERE event_type IN ('search', 'filter')
		  AND filters->'state' IS NOT NULL
		  AND created_at > NOW() - INTERVAL '%s'
		GROUP BY value
		ORDER BY count DESC
		LIMIT 10
	`, interval)

	rows, err = r.pool.Query(ctx, stateQuery)
	if err != nil {
		return nil, err
	}
	for rows.Next() {
		var stat models.FilterStat
		if err := rows.Scan(&stat.Value, &stat.Count); err != nil {
			rows.Close()
			return nil, err
		}
		stats.States = append(stats.States, stat)
	}
	rows.Close()

	return stats, nil
}

func (r *AnalyticsRepository) getClickStats(ctx context.Context, interval string) (*models.ClickStats, error) {
	stats := &models.ClickStats{
		ByPosition: []models.PositionStat{},
	}

	// Get total searches and clicks
	countQuery := fmt.Sprintf(`
		SELECT
			COALESCE(SUM(CASE WHEN event_type = 'search' THEN 1 ELSE 0 END), 0) as searches,
			COALESCE(SUM(CASE WHEN event_type = 'click' THEN 1 ELSE 0 END), 0) as clicks
		FROM search_events
		WHERE created_at > NOW() - INTERVAL '%s'
	`, interval)

	err := r.pool.QueryRow(ctx, countQuery).Scan(&stats.TotalSearches, &stats.TotalClicks)
	if err != nil {
		return nil, err
	}

	if stats.TotalSearches > 0 {
		stats.ClickThroughRate = float64(stats.TotalClicks) / float64(stats.TotalSearches) * 100
	}

	// Get clicks by position
	positionQuery := fmt.Sprintf(`
		SELECT result_position, COUNT(*) as clicks
		FROM search_events
		WHERE event_type = 'click'
		  AND result_position IS NOT NULL
		  AND created_at > NOW() - INTERVAL '%s'
		GROUP BY result_position
		ORDER BY result_position
		LIMIT 20
	`, interval)

	rows, err := r.pool.Query(ctx, positionQuery)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var stat models.PositionStat
		if err := rows.Scan(&stat.Position, &stat.Clicks); err != nil {
			return nil, err
		}
		stats.ByPosition = append(stats.ByPosition, stat)
	}

	return stats, rows.Err()
}

func (r *AnalyticsRepository) getEventCounts(ctx context.Context, interval string) (*models.EventCounts, error) {
	query := fmt.Sprintf(`
		SELECT
			COALESCE(SUM(CASE WHEN event_type = 'search' THEN 1 ELSE 0 END), 0) as searches,
			COALESCE(SUM(CASE WHEN event_type = 'filter' THEN 1 ELSE 0 END), 0) as filters,
			COALESCE(SUM(CASE WHEN event_type = 'click' THEN 1 ELSE 0 END), 0) as clicks,
			COALESCE(SUM(CASE WHEN event_type = 'page' THEN 1 ELSE 0 END), 0) as pages
		FROM search_events
		WHERE created_at > NOW() - INTERVAL '%s'
	`, interval)

	var counts models.EventCounts
	err := r.pool.QueryRow(ctx, query).Scan(&counts.Searches, &counts.Filters, &counts.Clicks, &counts.Pages)
	if err != nil {
		return nil, err
	}

	return &counts, nil
}
