package models

import (
	"time"
)

type Opportunity struct {
	ID                 int        `json:"id"`
	NoticeID           string     `json:"notice_id"`
	Title              string     `json:"title"`
	Description        *string    `json:"description,omitempty"`
	SolicitationNumber *string    `json:"solicitation_number,omitempty"`
	Type               *string    `json:"type,omitempty"`
	BaseType           *string    `json:"base_type,omitempty"`
	Department         *string    `json:"department,omitempty"`
	PostedDate         *time.Time `json:"posted_date,omitempty"`
	ResponseDeadline   *time.Time `json:"response_deadline,omitempty"`
	ArchiveDate        *time.Time `json:"archive_date,omitempty"`
	SetAsideCode       *string    `json:"set_aside_code,omitempty"`
	SetAsideDesc       *string    `json:"set_aside_description,omitempty"`
	NAICSCode          *string    `json:"naics_code,omitempty"`
	NAICSCodes         []string   `json:"naics_codes,omitempty"`
	ClassificationCode *string    `json:"classification_code,omitempty"`
	PopStreetAddress   *string    `json:"pop_street_address,omitempty"`
	PopCity            *string    `json:"pop_city,omitempty"`
	PopState           *string    `json:"pop_state,omitempty"`
	PopZip             *string    `json:"pop_zip,omitempty"`
	PopCountry         *string    `json:"pop_country,omitempty"`
	AwardNumber        *string    `json:"award_number,omitempty"`
	AwardAmount        *float64   `json:"award_amount,omitempty"`
	AwardeeName        *string    `json:"awardee_name,omitempty"`
	AwardeeUEI         *string    `json:"awardee_uei,omitempty"`
	AwardDate          *time.Time `json:"award_date,omitempty"`
	Active             bool       `json:"active"`
	UILink             *string    `json:"ui_link,omitempty"`
	ResourceLinks      []string   `json:"resource_links,omitempty"`
	DataSource         *string    `json:"data_source,omitempty"`
	CreatedAt          time.Time  `json:"created_at"`
	UpdatedAt          time.Time  `json:"updated_at"`
}

type OpportunityListItem struct {
	ID                 int        `json:"id"`
	NoticeID           string     `json:"notice_id"`
	Title              string     `json:"title"`
	Description        *string    `json:"description,omitempty"`
	SolicitationNumber *string    `json:"solicitation_number,omitempty"`
	Type               *string    `json:"type,omitempty"`
	Department         *string    `json:"department,omitempty"`
	PostedDate         *time.Time `json:"posted_date,omitempty"`
	ResponseDeadline   *time.Time `json:"response_deadline,omitempty"`
	SetAsideCode       *string    `json:"set_aside_code,omitempty"`
	SetAsideDesc       *string    `json:"set_aside_description,omitempty"`
	NAICSCode          *string    `json:"naics_code,omitempty"`
	PopState           *string    `json:"pop_state,omitempty"`
	Active             bool       `json:"active"`
}

type SearchParams struct {
	Query        string
	Types        []string
	SetAsides    []string
	NAICSCodes   []string
	States       []string
	PostedFrom   *time.Time
	PostedTo     *time.Time
	DeadlineFrom *time.Time
	DeadlineTo   *time.Time
	Sort         string
	Order        string
	Page         int
	Limit        int
}

type SearchResult struct {
	Opportunities []OpportunityListItem `json:"opportunities"`
	Total         int                   `json:"total"`
	Page          int                   `json:"page"`
	Limit         int                   `json:"limit"`
	TotalPages    int                   `json:"total_pages"`
}

type FilterOptions struct {
	Types     []FilterOption `json:"types"`
	SetAsides []FilterOption `json:"set_asides"`
	States    []FilterOption `json:"states"`
}

type FilterOption struct {
	Code  string `json:"code"`
	Label string `json:"label,omitempty"`
	Count int    `json:"count"`
}
