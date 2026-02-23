package models

import "time"

type SavedOpportunity struct {
	ID                 int        `json:"id"`
	UserID             int        `json:"user_id"`
	OpportunityID      int        `json:"opportunity_id"`
	NoticeID           string     `json:"notice_id"`
	SolicitationNumber *string    `json:"solicitation_number,omitempty"`
	Notes              *string    `json:"notes,omitempty"`
	LastNotifiedAt     time.Time  `json:"last_notified_at"`
	CreatedAt          time.Time  `json:"created_at"`
	UpdatedAt          time.Time  `json:"updated_at"`
}

type SavedOpportunityDetail struct {
	ID                 int        `json:"id"`
	OpportunityID      int        `json:"opportunity_id"`
	NoticeID           string     `json:"notice_id"`
	SolicitationNumber *string    `json:"solicitation_number,omitempty"`
	Notes              *string    `json:"notes,omitempty"`
	CreatedAt          time.Time  `json:"created_at"`
	Title              string     `json:"title"`
	Description        *string    `json:"description,omitempty"`
	Type               *string    `json:"type,omitempty"`
	Department         *string    `json:"department,omitempty"`
	PostedDate         *time.Time `json:"posted_date,omitempty"`
	ResponseDeadline   *time.Time `json:"response_deadline,omitempty"`
	SetAsideCode       *string    `json:"set_aside_code,omitempty"`
	SetAsideDesc       *string    `json:"set_aside_description,omitempty"`
	NAICSCode          *string    `json:"naics_code,omitempty"`
	PopState           *string    `json:"pop_state,omitempty"`
	Active             bool       `json:"active"`
	HasUpdates         bool       `json:"has_updates"`
}
