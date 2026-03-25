package models

import (
	"encoding/json"
	"time"
)

type SavedSearch struct {
	ID               int              `json:"id"`
	UserID           int              `json:"user_id"`
	Name             string           `json:"name"`
	Filters          json.RawMessage  `json:"filters"`
	AlertEnabled     bool             `json:"alert_enabled"`
	LastCheckedAt    *time.Time       `json:"last_checked_at,omitempty"`
	LastMatchCount   int              `json:"last_match_count"`
	TotalResultCount *int             `json:"total_result_count,omitempty"`
	CreatedAt        time.Time        `json:"created_at"`
	UpdatedAt        time.Time        `json:"updated_at"`
}

type CreateSavedSearchInput struct {
	Name         string          `json:"name"`
	Filters      json.RawMessage `json:"filters"`
	AlertEnabled *bool           `json:"alert_enabled,omitempty"`
}

type UpdateSavedSearchInput struct {
	Name         *string          `json:"name,omitempty"`
	Filters      *json.RawMessage `json:"filters,omitempty"`
	AlertEnabled *bool            `json:"alert_enabled,omitempty"`
}

type DayCount struct {
	Date  string `json:"date"`
	Count int    `json:"count"`
}

type HistoryTimelineResponse struct {
	Search          SavedSearch `json:"search"`
	Days            []DayCount  `json:"days"`
	TotalNew        int         `json:"total_new"`
	DaysWithMatches int         `json:"days_with_matches"`
}
