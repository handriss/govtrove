package reconcile

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/handriss/govtrove/pipeline/internal/parse"
)

var wsRun = regexp.MustCompile(`\s+`)

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
	MiddleTier string // API-only: intermediate org levels between SubTier and Office

	PopStreetAddress string
	PopCity          string
	PopState         string
	PopZip           string
	PopCountry       string
	PopCityCode      string
	PopStateCode     string
	PopCountryCode   string
	PopStateName     string `json:"-"`
	PopCountryName   string `json:"-"`

	OfficeCity    string
	OfficeState   string
	OfficeZip     string
	OfficeCountry string

	AwardNumber string
	AwardDate   *time.Time
	AwardAmount *float64
	Awardee     string
	AwardeeName string
	AwardeeUeiSAM        string
	AwardeeStreetAddress string
	AwardeeCity          string
	AwardeeCityCode      string
	AwardeeState         string
	AwardeeStateCode     string
	AwardeeCountry       string
	AwardeeCountryCode   string
	AwardeeZip           string

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

	// Metadata (excluded from content hash)
	DataSources string `json:"-"`
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

type ReconcileMismatch struct {
	FieldName string
	IssueType string
	CSVValue  string
	APIValue  string
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

		PostedDate:       parse.DateOnly(raw["PostedDate"]),
		ResponseDeadline: parse.DateOnly(raw["ResponseDeadLine"]),
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
		PopStateName:     LookupStateName(raw["PopState"]),
		PopCountryName:   LookupCountryName(raw["PopCountry"]),

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
// When both sources are present, compares fields that should be identical
// and reports mismatches. Prefers CSV for the merged result.
func ReconcileRecord(csv, api *Opportunity) (Opportunity, []ReconcileMismatch) {
	if csv == nil {
		opp := *api
		opp.DataSources = "api"
		return opp, []ReconcileMismatch{{
			FieldName: "source",
			IssueType: "missing_csv",
			APIValue:  api.NoticeID,
		}}
	}
	if api == nil {
		opp := *csv
		opp.DataSources = "csv"
		return opp, []ReconcileMismatch{{
			FieldName: "source",
			IssueType: "missing_api",
			CSVValue:  csv.NoticeID,
		}}
	}

	var mismatches []ReconcileMismatch
	normalize := func(s string) string {
		s = strings.TrimSpace(wsRun.ReplaceAllString(s, " "))
		s = strings.ReplaceAll(s, "U.S.", "US")
		return s
	}
	cmpStr := func(field, csvVal, apiVal string) {
		if csvVal == "" || apiVal == "" {
			return
		}
		if normalize(csvVal) != normalize(apiVal) {
			mismatches = append(mismatches, ReconcileMismatch{FieldName: field, IssueType: "field_mismatch", CSVValue: csvVal, APIValue: apiVal})
		}
	}
	cmpTime := func(field string, csvVal, apiVal *time.Time) {
		if csvVal == nil || apiVal == nil {
			return
		}
		if !csvVal.Equal(*apiVal) {
			mismatches = append(mismatches, ReconcileMismatch{
				FieldName: field, IssueType: "field_mismatch",
				CSVValue: csvVal.Format(time.DateOnly),
				APIValue: apiVal.Format(time.DateOnly),
			})
		}
	}
	cmpAmount := func(field string, csvVal, apiVal *float64) {
		if csvVal == nil || apiVal == nil {
			return
		}
		if *csvVal != *apiVal {
			mismatches = append(mismatches, ReconcileMismatch{
				FieldName: field, IssueType: "field_mismatch",
				CSVValue: fmt.Sprintf("%.2f", *csvVal),
				APIValue: fmt.Sprintf("%.2f", *apiVal),
			})
		}
	}
	cmpBool := func(field string, csvVal, apiVal bool) {
		if csvVal != apiVal {
			mismatches = append(mismatches, ReconcileMismatch{
				FieldName: field, IssueType: "field_mismatch",
				CSVValue: fmt.Sprintf("%t", csvVal),
				APIValue: fmt.Sprintf("%t", apiVal),
			})
		}
	}

	cmpStr("SolicitationNumber", csv.SolicitationNumber, api.SolicitationNumber)
	cmpStr("Title", csv.Title, api.Title)
	cmpStr("Type", csv.Type, api.Type)
	cmpStr("BaseType", csv.BaseType, api.BaseType)
	cmpStr("OrganizationType", csv.OrganizationType, api.OrganizationType)
	cmpTime("PostedDate", csv.PostedDate, api.PostedDate)
	cmpTime("ResponseDeadline", csv.ResponseDeadline, api.ResponseDeadline)
	cmpTime("ArchiveDate", csv.ArchiveDate, api.ArchiveDate)
	cmpStr("ArchiveType", csv.ArchiveType, api.ArchiveType)
	cmpBool("Active", csv.Active, api.Active)
	cmpStr("SetAsideCode", csv.SetAsideCode, api.SetAsideCode)
	cmpStr("SetAsideDescription", csv.SetAsideDescription, api.SetAsideDescription)
	cmpStr("NAICSCode", csv.NAICSCode, api.NAICSCode)
	cmpStr("ClassificationCode", csv.ClassificationCode, api.ClassificationCode)
	cmpStr("AwardNumber", csv.AwardNumber, api.AwardNumber)
	cmpTime("AwardDate", csv.AwardDate, api.AwardDate)
	cmpAmount("AwardAmount", csv.AwardAmount, api.AwardAmount)
	cmpStr("UILink", csv.UILink, api.UILink)

	// "Should match" fields — both sources populate, same underlying data
	cmpStr("PopStreetAddress", csv.PopStreetAddress, api.PopStreetAddress)
	cmpStr("PopCity", csv.PopCity, api.PopCity)
	cmpStr("PopZip", csv.PopZip, api.PopZip)
	cmpStr("OfficeCity", csv.OfficeCity, api.OfficeCity)
	cmpStr("OfficeState", csv.OfficeState, api.OfficeState)
	cmpStr("OfficeZip", csv.OfficeZip, api.OfficeZip)
	cmpStr("OfficeCountry", csv.OfficeCountry, api.OfficeCountry)
	cmpStr("PrimaryContactTitle", csv.PrimaryContactTitle, api.PrimaryContactTitle)
	cmpStr("PrimaryContactFullname", csv.PrimaryContactFullname, api.PrimaryContactFullname)
	cmpStr("PrimaryContactEmail", csv.PrimaryContactEmail, api.PrimaryContactEmail)
	cmpStr("PrimaryContactPhone", csv.PrimaryContactPhone, api.PrimaryContactPhone)
	cmpStr("PrimaryContactFax", csv.PrimaryContactFax, api.PrimaryContactFax)
	cmpStr("SecondaryContactTitle", csv.SecondaryContactTitle, api.SecondaryContactTitle)
	cmpStr("SecondaryContactFullname", csv.SecondaryContactFullname, api.SecondaryContactFullname)
	cmpStr("SecondaryContactEmail", csv.SecondaryContactEmail, api.SecondaryContactEmail)
	cmpStr("SecondaryContactPhone", csv.SecondaryContactPhone, api.SecondaryContactPhone)
	cmpStr("SecondaryContactFax", csv.SecondaryContactFax, api.SecondaryContactFax)

	// Org hierarchy — derived differently (CSV has explicit columns, API splits dot path)
	cmpStr("Department", csv.Department, api.Department)
	cmpStr("SubTier", csv.SubTier, api.SubTier)
	cmpStr("Office", csv.Office, api.Office)
	cmpStr("CGAC", csv.CGAC, api.CGAC)
	cmpStr("FPDSCode", csv.FPDSCode, api.FPDSCode)
	cmpStr("AACCode", csv.AACCode, api.AACCode)

	merged := *csv
	merged.DataSources = "csv+api"

	// Carry forward API-only awardee fields (no merging — keep both sides)
	merged.AwardeeName = api.AwardeeName
	merged.AwardeeUeiSAM = api.AwardeeUeiSAM
	merged.AwardeeStreetAddress = api.AwardeeStreetAddress
	merged.AwardeeCity = api.AwardeeCity
	merged.AwardeeCityCode = api.AwardeeCityCode
	merged.AwardeeState = api.AwardeeState
	merged.AwardeeStateCode = api.AwardeeStateCode
	merged.AwardeeCountry = api.AwardeeCountry
	merged.AwardeeCountryCode = api.AwardeeCountryCode
	merged.AwardeeZip = api.AwardeeZip

	// Carry forward API-only fields
	merged.MiddleTier = api.MiddleTier
	merged.DescriptionURL = api.DescriptionURL
	merged.PopCityCode = api.PopCityCode
	merged.PopStateCode = api.PopStateCode
	merged.PopCountryCode = api.PopCountryCode
	merged.FullParentPathName = api.FullParentPathName
	merged.FullParentPathCode = api.FullParentPathCode
	merged.AdditionalInfoLink = api.AdditionalInfoLink
	merged.ResourceLinks = api.ResourceLinks

	return merged, mismatches
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
