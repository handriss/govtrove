package models

import (
	"encoding/json"
	"time"
)

type UserUpdate struct {
	ID             string          `json:"id"`
	UserID         int             `json:"user_id"`
	UpdateType     string          `json:"update_type"`
	SourceID       *int            `json:"source_id,omitempty"`
	OpportunityIDs []int           `json:"opportunity_ids,omitempty"`
	Summary        string          `json:"summary"`
	Details        json.RawMessage `json:"details,omitempty"`
	IsRead         bool            `json:"is_read"`
	CreatedAt      time.Time       `json:"created_at"`
}

type UserUpdateCount struct {
	Unread int `json:"unread"`
	Total  int `json:"total"`
}
