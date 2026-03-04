package models

import (
	"encoding/json"
	"time"
)

type Notification struct {
	ID         string          `json:"id"`
	UserID     int             `json:"user_id"`
	UpdateType string          `json:"update_type"`
	SourceID   *int            `json:"source_id,omitempty"`
	GroupKey   *string         `json:"group_key,omitempty"`
	Details    json.RawMessage `json:"details"`
	IsRead     bool            `json:"is_read"`
	ExpiresAt  time.Time       `json:"expires_at"`
	CreatedAt  time.Time       `json:"created_at"`
}

type NotificationCount struct {
	Unread int `json:"unread"`
	Total  int `json:"total"`
}
