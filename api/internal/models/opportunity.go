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
	ClassificationCode *string    `json:"classification_code,omitempty"`
	PopStreetAddress   *string    `json:"pop_street_address,omitempty"`
	PopCity            *string    `json:"pop_city,omitempty"`
	PopState           *string    `json:"pop_state,omitempty"`
	PopZip             *string    `json:"pop_zip,omitempty"`
	PopCountry         *string    `json:"pop_country,omitempty"`
	PopCityCode        *string    `json:"pop_city_code,omitempty"`
	PopStateCode       *string    `json:"pop_state_code,omitempty"`
	PopCountryCode     *string    `json:"pop_country_code,omitempty"`
	PopStateName       *string    `json:"pop_state_name,omitempty"`
	PopCountryName     *string    `json:"pop_country_name,omitempty"`
	AwardNumber        *string    `json:"award_number,omitempty"`
	AwardAmount        *float64   `json:"award_amount,omitempty"`
	AwardeeName        *string    `json:"awardee_name,omitempty"`
	AwardeeUEI         *string    `json:"awardee_uei,omitempty"`
	AwardDate          *time.Time `json:"award_date,omitempty"`
	Active                  bool       `json:"active"`
	UILink                  *string    `json:"ui_link,omitempty"`
	ResourceLinks           []string   `json:"resource_links,omitempty"`
	DataSource              *string    `json:"data_source,omitempty"`
	SnapCSVID               *int64     `json:"snap_csv_id,omitempty"`
	SnapAPIID               *int64     `json:"snap_api_id,omitempty"`
	PrimaryContactTitle     *string    `json:"primary_contact_title,omitempty"`
	PrimaryContactFullname  *string    `json:"primary_contact_fullname,omitempty"`
	PrimaryContactEmail     *string    `json:"primary_contact_email,omitempty"`
	PrimaryContactPhone     *string    `json:"primary_contact_phone,omitempty"`
	PrimaryContactFax       *string    `json:"primary_contact_fax,omitempty"`
	SecondaryContactTitle    *string    `json:"secondary_contact_title,omitempty"`
	SecondaryContactFullname *string    `json:"secondary_contact_fullname,omitempty"`
	SecondaryContactEmail    *string    `json:"secondary_contact_email,omitempty"`
	SecondaryContactPhone    *string    `json:"secondary_contact_phone,omitempty"`
	SecondaryContactFax      *string    `json:"secondary_contact_fax,omitempty"`
	CreatedAt               time.Time  `json:"created_at"`
	UpdatedAt               time.Time  `json:"updated_at"`
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
	NAICSCodes    []string
	NAICSPrefix   string
	NAICSPrefixes []string
	PSCCodes      []string
	PSCPrefix     string
	PSCPrefixes   []string
	States       []string
	Department   string
	AgencyPaths  []string
	PostedFrom   *time.Time
	PostedTo     *time.Time
	DeadlineFrom *time.Time
	DeadlineTo           *time.Time
	SolicitationNumber   string
	PopCity              string
	Sort                 string
	Order        string
	Page         int
	Limit        int

	GeoStates []string
	GeoCities []string
}

type SearchResult struct {
	Opportunities []OpportunityListItem `json:"opportunities"`
	Total         int                   `json:"total"`
	Page          int                   `json:"page"`
	Limit         int                   `json:"limit"`
	TotalPages    int                   `json:"total_pages"`
	Suggestion    string                `json:"suggestion,omitempty"`
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

type FacetValue struct {
	Value string `json:"value"`
	Label string `json:"label,omitempty"`
	Count int    `json:"count"`
}

type FacetResult struct {
	Total  int                     `json:"total"`
	Facets map[string][]FacetValue `json:"facets"`
}

type SolicitationHistoryItem struct {
	ID               int           `json:"id"`
	NoticeID         string        `json:"notice_id"`
	Title            string        `json:"title"`
	Type             *string       `json:"type,omitempty"`
	BaseType         *string       `json:"base_type,omitempty"`
	PostedDate       *time.Time    `json:"posted_date,omitempty"`
	ResponseDeadline *time.Time    `json:"response_deadline,omitempty"`
	ArchiveDate      *time.Time    `json:"archive_date,omitempty"`
	AwardDate        *time.Time    `json:"award_date,omitempty"`
	AwardAmount      *float64      `json:"award_amount,omitempty"`
	AwardeeName      *string       `json:"awardee_name,omitempty"`
	SetAsideCode     *string       `json:"set_aside_code,omitempty"`
	Active           bool          `json:"active"`
	IsCurrent        bool          `json:"is_current"`
	Version          int           `json:"version"`
	ContactName      *string       `json:"contact_name,omitempty"`
	ResourceCount    int           `json:"resource_count"`
	Description      *string       `json:"-"`
	Changes          []FieldChange `json:"changes,omitempty"`
}

type FieldChange struct {
	FieldName string  `json:"field_name"`
	OldValue  *string `json:"old_value,omitempty"`
	NewValue  *string `json:"new_value,omitempty"`
}

// SAM.gov single-letter notice type codes → full names stored in the DB.
var noticeTypeCodeMap = map[string]string{
	"o": "Solicitation",
	"p": "Presolicitation",
	"k": "Combined Synopsis/Solicitation",
	"r": "Sources Sought",
	"s": "Special Notice",
	"g": "Sale of Surplus Property",
	"i": "Intent to Bundle",
	"a": "Award Notice",
	"u": "Justification and Approval (J&A)",
}

// NormalizeNoticeTypes converts SAM.gov single-letter codes to the full
// type names used in the database. Values that are already full names
// pass through unchanged.
func NormalizeNoticeTypes(types []string) []string {
	out := make([]string, len(types))
	for i, t := range types {
		if full, ok := noticeTypeCodeMap[t]; ok {
			out[i] = full
		} else {
			out[i] = t
		}
	}
	return out
}

type SolicitationHistory struct {
	SolicitationNumber string                    `json:"solicitation_number"`
	TotalNotices       int                       `json:"total_notices"`
	Notices            []SolicitationHistoryItem  `json:"notices"`
	Truncated          bool                      `json:"truncated"`
}
