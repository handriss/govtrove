package reconcile

import (
	"strings"

	"github.com/handriss/govtrove/pipeline/internal/parse"
	"github.com/handriss/govtrove/pipeline/internal/samgov"
)

// FromAPI maps a SAM.gov API OpportunityData to the canonical Opportunity struct.
// Returns data quality issues for unparseable or sentinel date values.
func FromAPI(d samgov.OpportunityData) (Opportunity, []DataQualityIssue) {
	var issues []DataQualityIssue

	archiveDate := parseDateField(d.ArchiveDate, "ArchiveDate", &issues)

	opp := Opportunity{
		NoticeID:           d.NoticeID,
		SolicitationNumber: d.SolicitationNumber,
		Title:              d.Title,
		Type:               d.Type,
		BaseType:           d.BaseType,
		OrganizationType:   d.OrganizationType,

		PostedDate:       parse.DateOnly(d.PostedDate),
		ResponseDeadline: parse.DateOnly(d.ResponseDeadLine),
		ArchiveDate:      archiveDate,
		ArchiveType:      d.ArchiveType,
		Active:           parse.Active(d.Active),

		NAICSCode:          d.NaicsCode,
		ClassificationCode: d.ClassificationCode,

		UILink: d.UILink,

		// API-only fields
		FullParentPathName: d.FullParentPathName,
		FullParentPathCode: d.FullParentPathCode,
		DescriptionURL:     d.Description,
		ResourceLinks:      d.ResourceLinks,
	}

	// Derive org hierarchy from dot-separated paths
	if d.FullParentPathName != "" {
		parts := splitAndTrim(d.FullParentPathName)
		if len(parts) >= 1 {
			opp.Department = parts[0]
		}
		if len(parts) >= 2 {
			opp.SubTier = parts[1]
		}
		if len(parts) >= 3 {
			opp.Office = parts[len(parts)-1]
		}
		if len(parts) > 3 {
			opp.MiddleTier = strings.Join(parts[2:len(parts)-1], ".")
		}
	}
	if d.FullParentPathCode != "" {
		parts := splitAndTrim(d.FullParentPathCode)
		if len(parts) >= 1 {
			opp.CGAC = parts[0]
		}
		if len(parts) >= 2 {
			opp.FPDSCode = parts[1]
		}
		if len(parts) >= 3 {
			opp.AACCode = parts[len(parts)-1]
		}
	}

	if d.TypeOfSetAside != nil {
		opp.SetAsideCode = *d.TypeOfSetAside
	}
	if d.TypeOfSetAsideDescription != nil {
		opp.SetAsideDescription = *d.TypeOfSetAsideDescription
	}
	if d.AdditionalInfoLink != nil {
		opp.AdditionalInfoLink = *d.AdditionalInfoLink
	}

	// Place of performance
	if d.PlaceOfPerformance != nil {
		pop := d.PlaceOfPerformance
		if pop.City != nil {
			opp.PopCity = pop.City.Name
			opp.PopCityCode = pop.City.Code
		}
		if pop.State != nil {
			opp.PopState = pop.State.Name
			opp.PopStateCode = pop.State.Code
		}
		if pop.Country != nil {
			opp.PopCountry = pop.Country.Name
			opp.PopCountryCode = pop.Country.Code
		}
		opp.PopZip = pop.Zip
		opp.PopStreetAddress = pop.StreetAddress
	}

	// Office address
	if d.OfficeAddress != nil {
		opp.OfficeCity = d.OfficeAddress.City
		opp.OfficeState = d.OfficeAddress.State
		opp.OfficeZip = d.OfficeAddress.Zipcode
		opp.OfficeCountry = d.OfficeAddress.CountryCode
	}

	// Contacts
	for _, c := range d.PointOfContact {
		switch strings.ToLower(c.Type) {
		case "primary":
			opp.PrimaryContactTitle = c.Title
			opp.PrimaryContactFullname = c.FullName
			opp.PrimaryContactEmail = c.Email
			opp.PrimaryContactPhone = c.Phone
			opp.PrimaryContactFax = c.Fax
		case "secondary":
			opp.SecondaryContactTitle = c.Title
			opp.SecondaryContactFullname = c.FullName
			opp.SecondaryContactEmail = c.Email
			opp.SecondaryContactPhone = c.Phone
			opp.SecondaryContactFax = c.Fax
		}
	}

	// Award info
	if d.Award != nil {
		opp.AwardNumber = d.Award.Number
		opp.AwardDate = parseDateField(d.Award.Date, "AwardDate", &issues)
		opp.AwardAmount = parse.Amount(d.Award.Amount)
		opp.AwardeeName = d.Award.Awardee.Name
		opp.AwardeeUeiSAM = d.Award.Awardee.UeiSAM

		if loc := d.Award.Awardee.Location; loc != nil {
			opp.AwardeeStreetAddress = loc.StreetAddress
			if loc.City != nil {
				opp.AwardeeCity = loc.City.Name
				opp.AwardeeCityCode = loc.City.Code
			}
			if loc.State != nil {
				opp.AwardeeState = loc.State.Name
				opp.AwardeeStateCode = loc.State.Code
			}
			if loc.Country != nil {
				opp.AwardeeCountry = loc.Country.Name
				opp.AwardeeCountryCode = loc.Country.Code
			}
			opp.AwardeeZip = loc.Zip
		}
	}

	return opp, issues
}

func splitAndTrim(s string) []string {
	parts := strings.Split(s, ".")
	for i := range parts {
		parts[i] = strings.TrimSpace(parts[i])
	}
	return parts
}
