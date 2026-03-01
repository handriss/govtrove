package reconcile

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"time"

	"github.com/handriss/govtrove/pipeline/internal/parse"
)

// Opportunity is the write model for upserting into the opportunities table.
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
	ArchiveDate      *time.Time
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
	PopCityCode      string
	PopStateCode     string
	PopCountryCode   string

	OfficeCity    string
	OfficeState   string
	OfficeZip     string
	OfficeCountry string

	AwardNumber string
	AwardDate   *time.Time
	AwardAmount *float64
	Awardee     string
	AwardeeName string

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

	// API-only fields (NULL when populated from CSV)
	FullParentPathName string
	FullParentPathCode string
	DescriptionURL     string
	AdditionalInfoLink string
	ResourceLinks      []string
}

// ContentHash returns a deterministic SHA-256 hash of content fields.
// Excludes Active (feed metadata, not content) to avoid spurious versions
// when the same notice_id appears in both active and archived CSV feeds.
func (o Opportunity) ContentHash() string {
	tmp := o
	tmp.Active = false
	b, _ := json.Marshal(tmp)
	h := sha256.Sum256(b)
	return hex.EncodeToString(h[:])
}

type DataQualityIssue struct {
	FieldName  string
	FieldValue string
	IssueType  string
}

// FromCSV maps a raw CSV row (map of header→value) into an Opportunity.
// Returns data quality issues for unparseable or sentinel date values.
func FromCSV(raw map[string]string) (Opportunity, []DataQualityIssue) {
	var issues []DataQualityIssue

	archiveDate := parseDateField(raw["ArchiveDate"], "ArchiveDate", &issues)
	awardDate := parseDateField(raw["AwardDate"], "AwardDate", &issues)

	opp := Opportunity{
		NoticeID:           raw["NoticeId"],
		SolicitationNumber: raw["Sol#"],
		Title:              raw["Title"],
		Description:        raw["Description"],
		Type:               raw["Type"],
		BaseType:           raw["BaseType"],
		OrganizationType:   raw["OrganizationType"],

		PostedDate:       parse.Date(raw["PostedDate"]),
		ResponseDeadline: parse.Date(raw["ResponseDeadLine"]),
		ArchiveDate:      archiveDate,
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
		AwardDate:   awardDate,
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

	return opp, issues
}

// ReconcileRecord merges a CSV and API opportunity for the same notice_id.
// Stub: prefers CSV when present. Real merge logic comes later.
func ReconcileRecord(csv, api *Opportunity) Opportunity {
	if csv != nil {
		return *csv
	}
	return *api
}

func parseDateField(rawValue, fieldName string, issues *[]DataQualityIssue) *time.Time {
	if rawValue == "" {
		return nil
	}
	t := parse.DateOnly(rawValue)
	if t == nil {
		*issues = append(*issues, DataQualityIssue{
			FieldName:  fieldName,
			FieldValue: rawValue,
			IssueType:  "unparseable_date",
		})
		return nil
	}
	if parse.IsSentinelDate(t) {
		*issues = append(*issues, DataQualityIssue{
			FieldName:  fieldName,
			FieldValue: rawValue,
			IssueType:  "sentinel_date",
		})
		return nil
	}
	return t
}
