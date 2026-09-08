package reconcile_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/handriss/govtrove/pipeline/internal/reconcile"
	"github.com/handriss/govtrove/pipeline/internal/samgov"
)

var _ = Describe("FromAPI", func() {
	Context("set-aside code normalisation", func() {
		setAside := func(raw string) string {
			opp, _ := reconcile.FromAPI(samgov.OpportunityData{
				NoticeID: "SA-001", Active: "Yes", TypeOfSetAside: &raw,
			})
			return opp.SetAsideCode
		}

		It("unwraps the JSON-array form SAM sometimes returns", func() {
			// Stored verbatim this made 178 rows invisible to set_aside=SBA.
			Expect(setAside(`["SBA"]`)).To(Equal("SBA"))
		})

		It("maps an empty array element to an empty code", func() {
			Expect(setAside(`[""]`)).To(Equal(""))
		})

		It("leaves an ordinary scalar code untouched", func() {
			Expect(setAside("SDVOSBC")).To(Equal("SDVOSBC"))
		})

		It("keeps the raw value when it is not parseable as an array", func() {
			Expect(setAside("[not json")).To(Equal("[not json"))
		})
	})

	Context("Place of Performance codes", func() {
		It("extracts both name and code from City, State, and Country", func() {
			d := samgov.OpportunityData{
				NoticeID: "POP-001",
				Active:   "Yes",
				PlaceOfPerformance: &samgov.PoP{
					StreetAddress: "100 Main St",
					City:          &samgov.NameCode{Code: "12345", Name: "Springfield"},
					State:         &samgov.NameCode{Code: "IL", Name: "Illinois"},
					Country:       &samgov.NameCode{Code: "USA", Name: "United States"},
					Zip:           "62701",
				},
			}
			opp, _ := reconcile.FromAPI(d)

			Expect(opp.PopCity).To(Equal("Springfield"))
			Expect(opp.PopCityCode).To(Equal("12345"))
			Expect(opp.PopState).To(Equal("Illinois"))
			Expect(opp.PopStateCode).To(Equal("IL"))
			Expect(opp.PopCountry).To(Equal("United States"))
			Expect(opp.PopCountryCode).To(Equal("USA"))
			Expect(opp.PopZip).To(Equal("62701"))
			Expect(opp.PopStreetAddress).To(Equal("100 Main St"))
		})

		It("leaves all PoP fields empty when PlaceOfPerformance is nil", func() {
			d := samgov.OpportunityData{
				NoticeID:           "POP-002",
				Active:             "Yes",
				PlaceOfPerformance: nil,
			}
			opp, _ := reconcile.FromAPI(d)

			Expect(opp.PopCity).To(BeEmpty())
			Expect(opp.PopCityCode).To(BeEmpty())
			Expect(opp.PopState).To(BeEmpty())
			Expect(opp.PopStateCode).To(BeEmpty())
			Expect(opp.PopCountry).To(BeEmpty())
			Expect(opp.PopCountryCode).To(BeEmpty())
		})

		It("handles partial PoP — state and country but no city", func() {
			d := samgov.OpportunityData{
				NoticeID: "POP-003",
				Active:   "Yes",
				PlaceOfPerformance: &samgov.PoP{
					City:    nil,
					State:   &samgov.NameCode{Code: "CA", Name: "California"},
					Country: &samgov.NameCode{Code: "USA", Name: "United States"},
				},
			}
			opp, _ := reconcile.FromAPI(d)

			Expect(opp.PopCity).To(BeEmpty())
			Expect(opp.PopCityCode).To(BeEmpty())
			Expect(opp.PopState).To(Equal("California"))
			Expect(opp.PopStateCode).To(Equal("CA"))
			Expect(opp.PopCountry).To(Equal("United States"))
			Expect(opp.PopCountryCode).To(Equal("USA"))
		})

		It("handles NameCode with empty code but populated name", func() {
			d := samgov.OpportunityData{
				NoticeID: "POP-004",
				Active:   "Yes",
				PlaceOfPerformance: &samgov.PoP{
					City: &samgov.NameCode{Code: "", Name: "Unknown City"},
				},
			}
			opp, _ := reconcile.FromAPI(d)

			Expect(opp.PopCity).To(Equal("Unknown City"))
			Expect(opp.PopCityCode).To(BeEmpty())
		})

		It("handles country-only PoP (common for overseas)", func() {
			d := samgov.OpportunityData{
				NoticeID: "POP-005",
				Active:   "Yes",
				PlaceOfPerformance: &samgov.PoP{
					Country: &samgov.NameCode{Code: "DEU", Name: "Germany"},
				},
			}
			opp, _ := reconcile.FromAPI(d)

			Expect(opp.PopCity).To(BeEmpty())
			Expect(opp.PopCityCode).To(BeEmpty())
			Expect(opp.PopState).To(BeEmpty())
			Expect(opp.PopStateCode).To(BeEmpty())
			Expect(opp.PopCountry).To(Equal("Germany"))
			Expect(opp.PopCountryCode).To(Equal("DEU"))
		})
	})

	Context("Award / Awardee mapping", func() {
		It("sets AwardeeName from award.awardee.name and leaves Awardee empty", func() {
			d := samgov.OpportunityData{
				NoticeID: "AWD-001",
				Active:   "Yes",
				Award: &samgov.Award{
					Number:  "W91QUZ-26-C-0001",
					Date:    "2026-01-20",
					Amount:  "1234567.89",
					Awardee: samgov.Awardee{Name: "ACME Corporation"},
				},
			}
			opp, _ := reconcile.FromAPI(d)

			Expect(opp.AwardeeName).To(Equal("ACME Corporation"))
			Expect(opp.Awardee).To(BeEmpty())
			Expect(opp.AwardNumber).To(Equal("W91QUZ-26-C-0001"))
		})

		It("leaves both Awardee and AwardeeName empty when Award is nil", func() {
			d := samgov.OpportunityData{
				NoticeID: "AWD-002",
				Active:   "Yes",
				Award:    nil,
			}
			opp, _ := reconcile.FromAPI(d)

			Expect(opp.Awardee).To(BeEmpty())
			Expect(opp.AwardeeName).To(BeEmpty())
			Expect(opp.AwardNumber).To(BeEmpty())
		})

		It("handles award with empty awardee name", func() {
			d := samgov.OpportunityData{
				NoticeID: "AWD-003",
				Active:   "Yes",
				Award: &samgov.Award{
					Number:  "FA8532-26-R-0042",
					Awardee: samgov.Awardee{Name: ""},
				},
			}
			opp, _ := reconcile.FromAPI(d)

			Expect(opp.AwardeeName).To(BeEmpty())
			Expect(opp.AwardNumber).To(Equal("FA8532-26-R-0042"))
		})
	})

	Context("CSV FromCSV awardee", func() {
		It("keeps the raw blob in Awardee and parses the name into AwardeeName", func() {
			raw := map[string]string{
				"NoticeId": "CSV-001",
				"Awardee":  "ACME CORP Springfield IL 62701 USA",
			}
			opp, issues := reconcile.FromCSV(raw)

			Expect(issues).To(BeEmpty())
			Expect(opp.Awardee).To(Equal("ACME CORP Springfield IL 62701 USA"))
			Expect(opp.AwardeeName).To(Equal("ACME CORP"))
		})

		It("leaves AwardeeName empty when the name has no legal suffix", func() {
			raw := map[string]string{
				"NoticeId": "CSV-002",
				"Awardee":  "OREGON STATE UNIVERSITY Corvallis OR 97331 USA",
			}
			opp, _ := reconcile.FromCSV(raw)

			Expect(opp.Awardee).To(Equal("OREGON STATE UNIVERSITY Corvallis OR 97331 USA"))
			Expect(opp.AwardeeName).To(BeEmpty())
		})
	})

	Context("Data quality issues from API dates", func() {
		It("flags sentinel ArchiveDate", func() {
			d := samgov.OpportunityData{
				NoticeID:    "DQ-001",
				Active:      "Yes",
				ArchiveDate: "1969-12-31",
			}
			opp, issues := reconcile.FromAPI(d)

			Expect(opp.ArchiveDate).To(BeNil())
			Expect(issues).To(HaveLen(1))
			Expect(issues[0].FieldName).To(Equal("ArchiveDate"))
			Expect(issues[0].IssueType).To(Equal("sentinel_date"))
		})

		It("flags unparseable ArchiveDate", func() {
			d := samgov.OpportunityData{
				NoticeID:    "DQ-002",
				Active:      "Yes",
				ArchiveDate: "not-a-date",
			}
			opp, issues := reconcile.FromAPI(d)

			Expect(opp.ArchiveDate).To(BeNil())
			Expect(issues).To(HaveLen(1))
			Expect(issues[0].FieldName).To(Equal("ArchiveDate"))
			Expect(issues[0].IssueType).To(Equal("unparseable_date"))
		})

		It("flags sentinel AwardDate", func() {
			d := samgov.OpportunityData{
				NoticeID: "DQ-003",
				Active:   "Yes",
				Award: &samgov.Award{
					Date: "1970-01-01",
				},
			}
			opp, issues := reconcile.FromAPI(d)

			Expect(opp.AwardDate).To(BeNil())
			Expect(issues).To(HaveLen(1))
			Expect(issues[0].FieldName).To(Equal("AwardDate"))
			Expect(issues[0].IssueType).To(Equal("sentinel_date"))
		})

		It("returns no issues for valid dates", func() {
			d := samgov.OpportunityData{
				NoticeID:    "DQ-004",
				Active:      "Yes",
				ArchiveDate: "2026-06-01",
				Award: &samgov.Award{
					Date: "2026-01-20",
				},
			}
			opp, issues := reconcile.FromAPI(d)

			Expect(opp.ArchiveDate).NotTo(BeNil())
			Expect(opp.AwardDate).NotTo(BeNil())
			Expect(issues).To(BeEmpty())
		})

		It("returns no issues when Award is nil and ArchiveDate is empty", func() {
			d := samgov.OpportunityData{
				NoticeID: "DQ-005",
				Active:   "Yes",
			}
			_, issues := reconcile.FromAPI(d)

			Expect(issues).To(BeEmpty())
		})
	})

	Context("ContentHash includes new fields", func() {
		It("changes when PopCityCode changes", func() {
			base := reconcile.Opportunity{NoticeID: "HASH-001", PopCity: "Springfield"}
			withCode := base
			withCode.PopCityCode = "12345"

			Expect(base.ContentHash()).NotTo(Equal(withCode.ContentHash()))
		})

		It("changes when PopStateCode changes", func() {
			base := reconcile.Opportunity{NoticeID: "HASH-002", PopState: "Illinois"}
			withCode := base
			withCode.PopStateCode = "IL"

			Expect(base.ContentHash()).NotTo(Equal(withCode.ContentHash()))
		})

		It("changes when PopCountryCode changes", func() {
			base := reconcile.Opportunity{NoticeID: "HASH-003", PopCountry: "United States"}
			withCode := base
			withCode.PopCountryCode = "USA"

			Expect(base.ContentHash()).NotTo(Equal(withCode.ContentHash()))
		})

		It("does not change when only the derived AwardeeName differs", func() {
			// AwardeeName is parsed out of Awardee, not sourced independently.
			// Hashing it would re-version every award notice the moment the parser
			// changes, recording a change the feed never made.
			base := reconcile.Opportunity{NoticeID: "HASH-004", Awardee: "ACME Corp Reston VA USA"}
			withName := base
			withName.AwardeeName = "ACME Corp"

			Expect(base.ContentHash()).To(Equal(withName.ContentHash()))
		})

		It("still changes when the underlying Awardee changes", func() {
			base := reconcile.Opportunity{NoticeID: "HASH-006", Awardee: "ACME Corp Reston VA USA"}
			other := base
			other.Awardee = "OTHER Corp Reston VA USA"

			Expect(base.ContentHash()).NotTo(Equal(other.ContentHash()))
		})

		It("still changes when the awardee UEI changes", func() {
			base := reconcile.Opportunity{NoticeID: "HASH-007"}
			withUEI := base
			withUEI.AwardeeUeiSAM = "ABC123DEF456"

			Expect(base.ContentHash()).NotTo(Equal(withUEI.ContentHash()))
		})

		It("does not change when only Active flag differs", func() {
			a := reconcile.Opportunity{NoticeID: "HASH-005", Active: true}
			b := reconcile.Opportunity{NoticeID: "HASH-005", Active: false}

			Expect(a.ContentHash()).To(Equal(b.ContentHash()))
		})
	})
})
