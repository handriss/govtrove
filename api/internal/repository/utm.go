package repository

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

type UTMRepository struct {
	pool *pgxpool.Pool
}

func NewUTMRepository(pool *pgxpool.Pool) *UTMRepository {
	return &UTMRepository{pool: pool}
}

type UTMVisit struct {
	UTMSource   *string `json:"utm_source,omitempty"`
	UTMMedium   *string `json:"utm_medium,omitempty"`
	UTMCampaign *string `json:"utm_campaign,omitempty"`
	LandingPage string  `json:"landing_page"`
	Origin      *string `json:"origin,omitempty"`
	Referrer    *string `json:"referrer,omitempty"`
	UserAgent   *string `json:"user_agent,omitempty"`
}

func (r *UTMRepository) CreateVisit(ctx context.Context, v *UTMVisit) error {
	_, err := r.pool.Exec(ctx, `
		INSERT INTO utm_visits (utm_source, utm_medium, utm_campaign, landing_page, origin, referrer, user_agent)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
	`, v.UTMSource, v.UTMMedium, v.UTMCampaign, v.LandingPage, v.Origin, v.Referrer, v.UserAgent)
	if err != nil {
		return fmt.Errorf("inserting utm visit: %w", err)
	}
	return nil
}

type CampaignSummary struct {
	Campaign      string `json:"campaign"`
	LandingVisits int    `json:"landing_visits"`
	AppVisits     int    `json:"app_visits"`
	TotalVisits   int    `json:"total_visits"`
	FirstVisit    string `json:"first_visit"`
	LastVisit     string `json:"last_visit"`
}

type SourceSummary struct {
	Source string `json:"source"`
	Medium string `json:"medium"`
	Count  int    `json:"count"`
}

type DailyCounts struct {
	Date  string `json:"date"`
	Count int    `json:"count"`
}

type CampaignSearchActivity struct {
	Campaign      string       `json:"campaign"`
	TotalSearches int          `json:"total_searches"`
	TotalViews    int          `json:"total_views"`
	TopQueries    []QueryCount `json:"top_queries"`
}

type QueryCount struct {
	Query string `json:"query"`
	Count int    `json:"count"`
}

type UTMAnalytics struct {
	TotalVisits    int                      `json:"total_visits"`
	Campaigns      []CampaignSummary        `json:"campaigns"`
	Sources        []SourceSummary          `json:"sources"`
	DailyVisits    []DailyCounts            `json:"daily_visits"`
	SearchActivity []CampaignSearchActivity `json:"search_activity"`
}

func (r *UTMRepository) GetAnalytics(ctx context.Context, since *time.Time) (*UTMAnalytics, error) {
	result := &UTMAnalytics{}

	// Total visits
	var totalQuery string
	var args []interface{}
	if since != nil {
		totalQuery = `SELECT COUNT(*) FROM utm_visits WHERE created_at >= $1`
		args = []interface{}{*since}
	} else {
		totalQuery = `SELECT COUNT(*) FROM utm_visits`
	}
	if err := r.pool.QueryRow(ctx, totalQuery, args...).Scan(&result.TotalVisits); err != nil {
		return nil, fmt.Errorf("counting utm visits: %w", err)
	}

	// Campaign summary
	campaignQuery := `
		SELECT
			COALESCE(utm_campaign, '(none)') AS campaign,
			COUNT(*) FILTER (WHERE origin = 'landing') AS landing_visits,
			COUNT(*) FILTER (WHERE origin = 'app') AS app_visits,
			COUNT(*) AS total_visits,
			MIN(created_at)::text AS first_visit,
			MAX(created_at)::text AS last_visit
		FROM utm_visits
	`
	if since != nil {
		campaignQuery += ` WHERE created_at >= $1`
	}
	campaignQuery += ` GROUP BY COALESCE(utm_campaign, '(none)') ORDER BY total_visits DESC`
	campaignRows, err := r.pool.Query(ctx, campaignQuery, args...)
	if err != nil {
		return nil, fmt.Errorf("querying campaign summary: %w", err)
	}
	defer campaignRows.Close()
	for campaignRows.Next() {
		var c CampaignSummary
		if err := campaignRows.Scan(&c.Campaign, &c.LandingVisits, &c.AppVisits, &c.TotalVisits, &c.FirstVisit, &c.LastVisit); err != nil {
			return nil, fmt.Errorf("scanning campaign summary: %w", err)
		}
		result.Campaigns = append(result.Campaigns, c)
	}

	// Source summary
	sourceQuery := `
		SELECT COALESCE(utm_source, '(none)'), COALESCE(utm_medium, '(none)'), COUNT(*)
		FROM utm_visits
	`
	if since != nil {
		sourceQuery += ` WHERE created_at >= $1`
	}
	sourceQuery += ` GROUP BY COALESCE(utm_source, '(none)'), COALESCE(utm_medium, '(none)') ORDER BY COUNT(*) DESC`
	sourceRows, err := r.pool.Query(ctx, sourceQuery, args...)
	if err != nil {
		return nil, fmt.Errorf("querying source summary: %w", err)
	}
	defer sourceRows.Close()
	for sourceRows.Next() {
		var s SourceSummary
		if err := sourceRows.Scan(&s.Source, &s.Medium, &s.Count); err != nil {
			return nil, fmt.Errorf("scanning source summary: %w", err)
		}
		result.Sources = append(result.Sources, s)
	}

	// Daily visits
	dailyQuery := `
		SELECT created_at::date::text AS day, COUNT(*)
		FROM utm_visits
	`
	if since != nil {
		dailyQuery += ` WHERE created_at >= $1`
	}
	dailyQuery += ` GROUP BY created_at::date ORDER BY created_at::date`
	dailyRows, err := r.pool.Query(ctx, dailyQuery, args...)
	if err != nil {
		return nil, fmt.Errorf("querying daily visits: %w", err)
	}
	defer dailyRows.Close()
	for dailyRows.Next() {
		var d DailyCounts
		if err := dailyRows.Scan(&d.Date, &d.Count); err != nil {
			return nil, fmt.Errorf("scanning daily visits: %w", err)
		}
		result.DailyVisits = append(result.DailyVisits, d)
	}

	// Search activity by campaign
	searchQuery := `
		SELECT
			COALESCE(utm_campaign, '(none)') AS campaign,
			COUNT(*) FILTER (WHERE event_type = 'search') AS total_searches,
			COUNT(*) FILTER (WHERE event_type = 'view') AS total_views
		FROM search_events
		WHERE utm_campaign IS NOT NULL
	`
	if since != nil {
		searchQuery += ` AND created_at >= $1`
	}
	searchQuery += ` GROUP BY COALESCE(utm_campaign, '(none)') ORDER BY total_searches DESC`
	searchRows, err := r.pool.Query(ctx, searchQuery, args...)
	if err != nil {
		return nil, fmt.Errorf("querying search activity: %w", err)
	}
	defer searchRows.Close()
	for searchRows.Next() {
		var sa CampaignSearchActivity
		if err := searchRows.Scan(&sa.Campaign, &sa.TotalSearches, &sa.TotalViews); err != nil {
			return nil, fmt.Errorf("scanning search activity: %w", err)
		}
		result.SearchActivity = append(result.SearchActivity, sa)
	}

	// Top queries per campaign
	for i := range result.SearchActivity {
		campaign := result.SearchActivity[i].Campaign
		topQuery := `
			SELECT query, COUNT(*) AS cnt
			FROM search_events
			WHERE utm_campaign = $1 AND event_type = 'search' AND query IS NOT NULL
		`
		topArgs := []interface{}{campaign}
		if since != nil {
			topQuery += ` AND created_at >= $2`
			topArgs = append(topArgs, *since)
		}
		topQuery += ` GROUP BY query ORDER BY cnt DESC LIMIT 10`
		topRows, err := r.pool.Query(ctx, topQuery, topArgs...)
		if err != nil {
			return nil, fmt.Errorf("querying top queries: %w", err)
		}
		defer topRows.Close()
		for topRows.Next() {
			var q QueryCount
			if err := topRows.Scan(&q.Query, &q.Count); err != nil {
				return nil, fmt.Errorf("scanning top queries: %w", err)
			}
			result.SearchActivity[i].TopQueries = append(result.SearchActivity[i].TopQueries, q)
		}
	}

	return result, nil
}
