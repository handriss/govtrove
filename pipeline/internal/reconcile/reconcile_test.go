package reconcile_test

import (
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/handriss/govtrove/pipeline/internal/reconcile"
)

var _ = Describe("FromCSV", func() {
	Context("with valid dates", func() {
		It("parses ArchiveDate and AwardDate correctly", func() {
			raw := map[string]string{
				"NoticeId":    "abc123",
				"ArchiveDate": "2026-02-15",
				"AwardDate":   "2025-11-20",
			}
			opp, issues := reconcile.FromCSV(raw)

			Expect(issues).To(BeEmpty())
			Expect(opp.ArchiveDate).NotTo(BeNil())
			Expect(opp.ArchiveDate.Equal(time.Date(2026, 2, 15, 0, 0, 0, 0, time.UTC))).To(BeTrue())
			Expect(opp.AwardDate).NotTo(BeNil())
			Expect(opp.AwardDate.Equal(time.Date(2025, 11, 20, 0, 0, 0, 0, time.UTC))).To(BeTrue())
		})
	})

	Context("with empty dates", func() {
		It("returns nil dates and no issues", func() {
			raw := map[string]string{
				"NoticeId":    "abc123",
				"ArchiveDate": "",
				"AwardDate":   "",
			}
			opp, issues := reconcile.FromCSV(raw)

			Expect(issues).To(BeEmpty())
			Expect(opp.ArchiveDate).To(BeNil())
			Expect(opp.AwardDate).To(BeNil())
		})
	})

	Context("when ArchiveDate is a Unix epoch sentinel (1969-12-31)", func() {
		It("sets the date to nil and records a data quality issue", func() {
			raw := map[string]string{
				"NoticeId":    "abc123",
				"ArchiveDate": "1969-12-31",
				"AwardDate":   "",
			}
			opp, issues := reconcile.FromCSV(raw)

			Expect(opp.ArchiveDate).To(BeNil())
			Expect(issues).To(HaveLen(1))
			Expect(issues[0].IssueType).To(Equal("sentinel_date"))
			Expect(issues[0].FieldName).To(Equal("ArchiveDate"))
			Expect(issues[0].FieldValue).To(Equal("1969-12-31"))
		})
	})

	Context("with an unparseable date", func() {
		It("sets the date to nil and records an unparseable_date issue", func() {
			raw := map[string]string{
				"NoticeId":    "abc123",
				"ArchiveDate": "not-a-date",
				"AwardDate":   "",
			}
			opp, issues := reconcile.FromCSV(raw)

			Expect(opp.ArchiveDate).To(BeNil())
			Expect(issues).To(HaveLen(1))
			Expect(issues[0].IssueType).To(Equal("unparseable_date"))
		})
	})

	Context("with a midnight timestamp", func() {
		It("truncates to date-only", func() {
			raw := map[string]string{
				"NoticeId":    "abc123",
				"ArchiveDate": "",
				"AwardDate":   "2021-02-18T00:00:00",
			}
			opp, issues := reconcile.FromCSV(raw)

			Expect(issues).To(BeEmpty())
			Expect(opp.AwardDate).NotTo(BeNil())
			Expect(opp.AwardDate.Equal(time.Date(2021, 2, 18, 0, 0, 0, 0, time.UTC))).To(BeTrue())
		})
	})

	Context("with multiple bad dates", func() {
		It("records an issue for each bad date field", func() {
			raw := map[string]string{
				"NoticeId":    "abc123",
				"ArchiveDate": "1969-12-31",
				"AwardDate":   "garbage",
			}
			opp, issues := reconcile.FromCSV(raw)

			Expect(opp.ArchiveDate).To(BeNil())
			Expect(opp.AwardDate).To(BeNil())
			Expect(issues).To(HaveLen(2))
		})
	})

	Context("ReconcileRecord", func() {
		It("returns CSV version when both are present", func() {
			csv := &reconcile.Opportunity{NoticeID: "REC-001", Title: "CSV Title"}
			api := &reconcile.Opportunity{NoticeID: "REC-001", Title: "API Title"}

			result, mismatches := reconcile.ReconcileRecord(csv, api)

			Expect(result.Title).To(Equal("CSV Title"))
			Expect(mismatches).To(HaveLen(1))
			Expect(mismatches[0].FieldName).To(Equal("Title"))
			Expect(mismatches[0].IssueType).To(Equal("field_mismatch"))
			Expect(mismatches[0].CSVValue).To(Equal("CSV Title"))
			Expect(mismatches[0].APIValue).To(Equal("API Title"))
		})

		It("reports missing_api when only CSV is present", func() {
			csv := &reconcile.Opportunity{NoticeID: "REC-002", Title: "CSV Only"}

			result, mismatches := reconcile.ReconcileRecord(csv, nil)

			Expect(result.Title).To(Equal("CSV Only"))
			Expect(mismatches).To(HaveLen(1))
			Expect(mismatches[0].IssueType).To(Equal("missing_api"))
			Expect(mismatches[0].CSVValue).To(Equal("REC-002"))
		})

		It("reports missing_csv when only API is present", func() {
			api := &reconcile.Opportunity{NoticeID: "REC-003", Title: "API Only"}

			result, mismatches := reconcile.ReconcileRecord(nil, api)

			Expect(result.Title).To(Equal("API Only"))
			Expect(mismatches).To(HaveLen(1))
			Expect(mismatches[0].IssueType).To(Equal("missing_csv"))
			Expect(mismatches[0].APIValue).To(Equal("REC-003"))
		})

		It("reports no mismatches when fields match", func() {
			csv := &reconcile.Opportunity{NoticeID: "REC-004", Title: "Same", Type: "Solicitation", NAICSCode: "541511"}
			api := &reconcile.Opportunity{NoticeID: "REC-004", Title: "Same", Type: "Solicitation", NAICSCode: "541511"}

			_, mismatches := reconcile.ReconcileRecord(csv, api)

			Expect(mismatches).To(BeEmpty())
		})

		It("skips comparison when one side is empty", func() {
			csv := &reconcile.Opportunity{NoticeID: "REC-005", NAICSCode: "541511"}
			api := &reconcile.Opportunity{NoticeID: "REC-005", NAICSCode: ""}

			_, mismatches := reconcile.ReconcileRecord(csv, api)

			Expect(mismatches).To(BeEmpty())
		})

		It("reports multiple mismatches across fields", func() {
			posted := time.Date(2026, 1, 15, 0, 0, 0, 0, time.UTC)
			postedDiff := time.Date(2026, 1, 16, 0, 0, 0, 0, time.UTC)
			csv := &reconcile.Opportunity{NoticeID: "REC-006", Title: "A", Type: "X", PostedDate: &posted}
			api := &reconcile.Opportunity{NoticeID: "REC-006", Title: "B", Type: "Y", PostedDate: &postedDiff}

			_, mismatches := reconcile.ReconcileRecord(csv, api)

			fields := make(map[string]bool)
			for _, m := range mismatches {
				fields[m.FieldName] = true
			}
			Expect(fields).To(HaveKey("Title"))
			Expect(fields).To(HaveKey("Type"))
			Expect(fields).To(HaveKey("PostedDate"))
		})
	})

	Context("field mapping", func() {
		It("maps all CSV columns to the correct Opportunity fields", func() {
			raw := map[string]string{
				"NoticeId":                "NOTICE-001",
				"Sol#":                    "SOL-123",
				"Title":                   "Test Opportunity",
				"Description":             "A test description",
				"Type":                    "Solicitation",
				"BaseType":                "Award Notice",
				"OrganizationType":        "DoD",
				"PostedDate":              "2026-01-15",
				"ResponseDeadLine":        "2026-03-01",
				"ArchiveDate":             "2026-06-01",
				"ArchiveType":             "manual",
				"Active":                  "Yes",
				"SetASideCode":            "SBA",
				"SetASide":                "Total Small Business",
				"NaicsCode":               "541511",
				"ClassificationCode":      "D301",
				"Department/Ind.Agency":   "DoD",
				"Sub-Tier":                "Army",
				"Office":                  "ACC-APG",
				"CGAC":                    "097",
				"FPDS Code":               "XXX",
				"AAC Code":                "AAC1",
				"PopStreetAddress":        "123 Main St",
				"PopCity":                 "Bethesda",
				"PopState":                "MD",
				"PopZip":                  "20814",
				"PopCountry":              "USA",
				"City":                    "Pentagon City",
				"State":                   "VA",
				"ZipCode":                 "22202",
				"CountryCode":             "US",
				"AwardNumber":             "W91QUZ-26-C-0001",
				"AwardDate":               "2026-01-20",
				"Award$":                  "$1,234,567.89",
				"Awardee":                 "ACME Corp",
				"PrimaryContactTitle":     "Mr.",
				"PrimaryContactFullname":  "John Doe",
				"PrimaryContactEmail":     "john@example.com",
				"PrimaryContactPhone":     "555-1234",
				"PrimaryContactFax":       "555-5678",
				"SecondaryContactTitle":   "Ms.",
				"SecondaryContactFullname": "Jane Smith",
				"SecondaryContactEmail":   "jane@example.com",
				"SecondaryContactPhone":   "555-9012",
				"SecondaryContactFax":     "555-3456",
				"Link":                    "https://sam.gov/opp/abc123",
			}
			opp, issues := reconcile.FromCSV(raw)

			Expect(issues).To(BeEmpty())
			Expect(opp.NoticeID).To(Equal("NOTICE-001"))
			Expect(opp.SolicitationNumber).To(Equal("SOL-123"))
			Expect(opp.Title).To(Equal("Test Opportunity"))
			Expect(opp.Description).To(Equal("A test description"))
			Expect(opp.Type).To(Equal("Solicitation"))
			Expect(opp.BaseType).To(Equal("Award Notice"))
			Expect(opp.OrganizationType).To(Equal("DoD"))
			Expect(opp.Active).To(BeTrue())
			Expect(opp.SetAsideCode).To(Equal("SBA"))
			Expect(opp.SetAsideDescription).To(Equal("Total Small Business"))
			Expect(opp.NAICSCode).To(Equal("541511"))
			Expect(opp.ClassificationCode).To(Equal("D301"))
			Expect(opp.Department).To(Equal("DoD"))
			Expect(opp.SubTier).To(Equal("Army"))
			Expect(opp.Office).To(Equal("ACC-APG"))
			Expect(opp.CGAC).To(Equal("097"))
			Expect(opp.FPDSCode).To(Equal("XXX"))
			Expect(opp.AACCode).To(Equal("AAC1"))
			Expect(opp.PopStreetAddress).To(Equal("123 Main St"))
			Expect(opp.PopCity).To(Equal("Bethesda"))
			Expect(opp.PopState).To(Equal("MD"))
			Expect(opp.PopZip).To(Equal("20814"))
			Expect(opp.PopCountry).To(Equal("USA"))
			Expect(opp.OfficeCity).To(Equal("Pentagon City"))
			Expect(opp.OfficeState).To(Equal("VA"))
			Expect(opp.OfficeZip).To(Equal("22202"))
			Expect(opp.OfficeCountry).To(Equal("US"))
			Expect(opp.AwardNumber).To(Equal("W91QUZ-26-C-0001"))
			Expect(opp.AwardAmount).NotTo(BeNil())
			Expect(*opp.AwardAmount).To(BeNumerically("~", 1234567.89, 0.01))
			Expect(opp.Awardee).To(Equal("ACME Corp"))
			Expect(opp.PrimaryContactTitle).To(Equal("Mr."))
			Expect(opp.PrimaryContactFullname).To(Equal("John Doe"))
			Expect(opp.PrimaryContactEmail).To(Equal("john@example.com"))
			Expect(opp.PrimaryContactPhone).To(Equal("555-1234"))
			Expect(opp.PrimaryContactFax).To(Equal("555-5678"))
			Expect(opp.SecondaryContactTitle).To(Equal("Ms."))
			Expect(opp.SecondaryContactFullname).To(Equal("Jane Smith"))
			Expect(opp.SecondaryContactEmail).To(Equal("jane@example.com"))
			Expect(opp.SecondaryContactPhone).To(Equal("555-9012"))
			Expect(opp.SecondaryContactFax).To(Equal("555-3456"))
			Expect(opp.UILink).To(Equal("https://sam.gov/opp/abc123"))
		})
	})
})
