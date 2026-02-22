package models

import (
	"encoding/json"
	"time"
)

type SavedSearch struct {
	ID        int              `json:"id"`
	UserID    int              `json:"user_id"`
	Name      string           `json:"name"`
	Filters   json.RawMessage  `json:"filters"`
	CreatedAt time.Time        `json:"created_at"`
	UpdatedAt time.Time        `json:"updated_at"`
}

type CreateSavedSearchInput struct {
	Name    string          `json:"name"`
	Filters json.RawMessage `json:"filters"`
}

type UpdateSavedSearchInput struct {
	Name    *string          `json:"name,omitempty"`
	Filters *json.RawMessage `json:"filters,omitempty"`
}
