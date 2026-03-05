package models

import (
	"time"
)

type SearchEvent struct {
	ID            int                    `json:"id"`
	EventType     string                 `json:"event_type"`
	UserID        *int                   `json:"user_id,omitempty"`
	Query         *string                `json:"query,omitempty"`
	Filters       map[string]interface{} `json:"filters,omitempty"`
	SortBy        *string                `json:"sort_by,omitempty"`
	Page          *int                   `json:"page,omitempty"`
	TotalResults  *int                   `json:"total_results,omitempty"`
	OpportunityID *int                   `json:"opportunity_id,omitempty"`
	DurationMs    *int                   `json:"duration_ms,omitempty"`
	CreatedAt     time.Time              `json:"created_at"`
	UserAgent     *string                `json:"user_agent,omitempty"`
	Referer       *string                `json:"referer,omitempty"`
	UtmCampaign   *string                `json:"utm_campaign,omitempty"`
}
