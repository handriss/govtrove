package reconcile

import (
	"time"

	"github.com/handriss/govtrove/jobs/internal/parse"
)

// Opportunity is the write model for upserting into the opportunities table.
// Currently populated from CSV only. Will later merge CSV + API sources.
type Opportunity struct {
	NoticeID           string
	SolicitationNumber string
	Title              string
	Description        string
	Type               string
	BaseType           string
	OrganizationType   string

	PostedDate       *time.Time
	ResponseDeadline *time.Time
	ArchiveDate      string // DATE, not TIMESTAMPTZ
	ArchiveType      string
	Active           bool

	SetAsideCode        string
	SetAsideDescription string
	NAICSCode           string
	ClassificationCode  string

	Department string
	SubTier    string
	Office     string
	CGAC       string
	FPDSCode   string
	AACCode    string

	PopStreetAddress string
	PopCity          string
	PopState         string
	PopZip           string
	PopCountry       string

	OfficeCity    string
	OfficeState   string
	OfficeZip     string
	OfficeCountry string

	AwardNumber string
	AwardDate   string
	AwardAmount *float64
	Awardee     string

	PrimaryContactTitle    string
	PrimaryContactFullname string
	PrimaryContactEmail    string
	PrimaryContactPhone    string
	PrimaryContactFax      string

	SecondaryContactTitle    string
	SecondaryContactFullname string
	SecondaryContactEmail    string
	SecondaryContactPhone    string
	SecondaryContactFax      string

	UILink string
}

// FromCSV maps a raw CSV row (map of header→value) into an Opportunity.
// This is the identity transform — future versions will merge CSV + API.
func FromCSV(raw map[string]string) Opportunity {
	return Opportunity{
		NoticeID:           raw["NoticeId"],
		SolicitationNumber: raw["Sol#"],
		Title:              raw["Title"],
		Description:        raw["Description"],
		Type:               raw["Type"],
		BaseType:           raw["BaseType"],
		OrganizationType:   raw["OrganizationType"],

		PostedDate:       parse.Date(raw["PostedDate"]),
		ResponseDeadline: parse.Date(raw["ResponseDeadLine"]),
		ArchiveDate:      raw["ArchiveDate"],
		ArchiveType:      raw["ArchiveType"],
		Active:           parse.Active(raw["Active"]),

		SetAsideCode:        raw["SetASideCode"],
		SetAsideDescription: raw["SetASide"],
		NAICSCode:           raw["NaicsCode"],
		ClassificationCode:  raw["ClassificationCode"],

		Department: raw["Department/Ind.Agency"],
		SubTier:    raw["Sub-Tier"],
		Office:     raw["Office"],
		CGAC:       raw["CGAC"],
		FPDSCode:   raw["FPDS Code"],
		AACCode:    raw["AAC Code"],

		PopStreetAddress: raw["PopStreetAddress"],
		PopCity:          raw["PopCity"],
		PopState:         raw["PopState"],
		PopZip:           raw["PopZip"],
		PopCountry:       raw["PopCountry"],

		OfficeCity:    raw["City"],
		OfficeState:   raw["State"],
		OfficeZip:     raw["ZipCode"],
		OfficeCountry: raw["CountryCode"],

		AwardNumber: raw["AwardNumber"],
		AwardDate:   raw["AwardDate"],
		AwardAmount: parse.Amount(raw["Award$"]),
		Awardee:     raw["Awardee"],

		PrimaryContactTitle:    raw["PrimaryContactTitle"],
		PrimaryContactFullname: raw["PrimaryContactFullname"],
		PrimaryContactEmail:    raw["PrimaryContactEmail"],
		PrimaryContactPhone:    raw["PrimaryContactPhone"],
		PrimaryContactFax:      raw["PrimaryContactFax"],

		SecondaryContactTitle:    raw["SecondaryContactTitle"],
		SecondaryContactFullname: raw["SecondaryContactFullname"],
		SecondaryContactEmail:    raw["SecondaryContactEmail"],
		SecondaryContactPhone:    raw["SecondaryContactPhone"],
		SecondaryContactFax:      raw["SecondaryContactFax"],

		UILink: raw["Link"],
	}
}
