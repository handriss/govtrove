package reconcile

import (
	"strings"

	"github.com/handriss/govtrove/pipeline/internal/parse"
	"github.com/handriss/govtrove/pipeline/internal/samgov"
)

// FromAPI maps a SAM.gov API OpportunityData to the canonical Opportunity struct.
func FromAPI(d samgov.OpportunityData) Opportunity {
	opp := Opportunity{
		NoticeID:           d.NoticeID,
		SolicitationNumber: d.SolicitationNumber,
		Title:              d.Title,
		Type:               d.Type,
		BaseType:           d.BaseType,
		OrganizationType:   d.OrganizationType,

		PostedDate:       parse.Date(d.PostedDate),
		ResponseDeadline: parse.Date(d.ResponseDeadLine),
		ArchiveDate:      parse.DateOnly(d.ArchiveDate),
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

	if d.TypeOfSetAside != nil {
		opp.SetAsideCode = *d.TypeOfSetAside
	}
	if d.TypeOfSetAsideDescription != nil {
		opp.SetAsideDescription = *d.TypeOfSetAsideDescription
	}
	if d.AdditionalInfoLink != nil {
		opp.AdditionalInfoLink = *d.AdditionalInfoLink
	}

	// Derive department/sub_tier/office from fullParentPathName (dot-separated hierarchy)
	if d.FullParentPathName != "" {
		parts := strings.Split(d.FullParentPathName, ".")
		for i := range parts {
			parts[i] = strings.TrimSpace(parts[i])
		}
		if len(parts) >= 1 {
			opp.Department = parts[0]
		}
		if len(parts) >= 2 {
			opp.SubTier = parts[1]
		}
		if len(parts) >= 3 {
			opp.Office = parts[len(parts)-1]
		}
	}

	// Derive org codes from fullParentPathCode
	if d.FullParentPathCode != "" {
		parts := strings.Split(d.FullParentPathCode, ".")
		if len(parts) >= 1 {
			opp.CGAC = strings.TrimSpace(parts[0])
		}
		if len(parts) >= 2 {
			opp.FPDSCode = strings.TrimSpace(parts[1])
		}
		if len(parts) >= 3 {
			opp.AACCode = strings.TrimSpace(parts[len(parts)-1])
		}
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
		opp.AwardDate = parse.DateOnly(d.Award.Date)
		opp.AwardAmount = parse.Amount(d.Award.Amount)
		opp.AwardeeName = d.Award.Awardee.Name
	}

	return opp
}
