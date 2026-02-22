package models

import "time"

type SavedOpportunity struct {
	ID            int       `json:"id"`
	UserID        int       `json:"user_id"`
	OpportunityID int       `json:"opportunity_id"`
	CreatedAt     time.Time `json:"created_at"`
}
