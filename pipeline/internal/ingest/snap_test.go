package ingest_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/handriss/govtrove/pipeline/internal/database"
	"github.com/handriss/govtrove/pipeline/internal/ingest"
)

var _ = Describe("ExtractSnapCSVRow", func() {
	Context("with a fully populated CSV row", func() {
		var row database.SnapCSVRow

		BeforeEach(func() {
			raw := map[string]string{
				"NoticeId":             "NOTICE-001",
				"Sol#":                 "SOL-123",
				"Title":                "Test Opportunity",
				"Type":                 "Solicitation",
				"BaseType":             "Award Notice",
				"PostedDate":           "2026-01-15",
				"ResponseDeadLine":     "2026-03-01",
				"ArchiveDate":          "2026-06-01",
				"ArchiveType":          "manual",
				"SetASideCode":         "SBA",
				"NaicsCode":            "541511",
				"ClassificationCode":   "D301",
				"Active":               "Yes",
				"Department/Ind.Agency": "DoD",
				"Sub-Tier":             "Army",
				"Office":               "ACC-APG",
				"CGAC":                 "097",
				"FPDS Code":            "XXX",
				"AAC Code":             "AAC1",
				"AwardNumber":          "W91QUZ-26-C-0001",
				"AwardDate":            "2026-01-20",
				"Award$":               "$1,234.56",
			}
			row = ingest.ExtractSnapCSVRow(raw)
		})

		It("maps string fields correctly", func() {
			Expect(row.NoticeID).To(Equal("NOTICE-001"))
			Expect(row.SolicitationNumber).To(Equal("SOL-123"))
			Expect(row.Title).To(Equal("Test Opportunity"))
			Expect(row.Type).To(Equal("Solicitation"))
			Expect(row.BaseType).To(Equal("Award Notice"))
			Expect(row.ArchiveDate).To(Equal("2026-06-01"))
			Expect(row.ArchiveType).To(Equal("manual"))
			Expect(row.SetAsideCode).To(Equal("SBA"))
			Expect(row.NAICSCode).To(Equal("541511"))
			Expect(row.ClassificationCode).To(Equal("D301"))
			Expect(row.Department).To(Equal("DoD"))
			Expect(row.SubTier).To(Equal("Army"))
			Expect(row.Office).To(Equal("ACC-APG"))
			Expect(row.CGAC).To(Equal("097"))
			Expect(row.FPDSCode).To(Equal("XXX"))
			Expect(row.AACCode).To(Equal("AAC1"))
			Expect(row.AwardNumber).To(Equal("W91QUZ-26-C-0001"))
			Expect(row.AwardDate).To(Equal("2026-01-20"))
		})

		It("parses dates via samgov.ParseDate", func() {
			Expect(row.PostedDate).NotTo(BeNil())
			Expect(row.PostedDate.Year()).To(Equal(2026))
			Expect(row.PostedDate.Month().String()).To(Equal("January"))
			Expect(row.PostedDate.Day()).To(Equal(15))

			Expect(row.ResponseDeadline).NotTo(BeNil())
			Expect(row.ResponseDeadline.Year()).To(Equal(2026))
			Expect(row.ResponseDeadline.Month().String()).To(Equal("March"))
		})

		It("parses Active flag", func() {
			Expect(row.Active).To(BeTrue())
		})

		It("parses AwardAmount", func() {
			Expect(row.AwardAmount).NotTo(BeNil())
			Expect(*row.AwardAmount).To(BeNumerically("~", 1234.56, 0.01))
		})

		It("preserves raw data map", func() {
			Expect(row.RawData).To(HaveKey("NoticeId"))
			Expect(row.RawData["NoticeId"]).To(Equal("NOTICE-001"))
			Expect(row.RawData).To(HaveKey("Sol#"))
		})

		It("computes a deterministic content hash", func() {
			Expect(row.ContentHash).NotTo(BeEmpty())
			Expect(row.ContentHash).To(HaveLen(64)) // SHA-256 hex
		})
	})

	Context("with minimal fields", func() {
		It("handles missing optional fields gracefully", func() {
			raw := map[string]string{
				"NoticeId": "NOTICE-002",
				"Active":   "No",
			}
			row := ingest.ExtractSnapCSVRow(raw)

			Expect(row.NoticeID).To(Equal("NOTICE-002"))
			Expect(row.Active).To(BeFalse())
			Expect(row.PostedDate).To(BeNil())
			Expect(row.ResponseDeadline).To(BeNil())
			Expect(row.AwardAmount).To(BeNil())
			Expect(row.SolicitationNumber).To(BeEmpty())
			Expect(row.ContentHash).NotTo(BeEmpty())
		})
	})

	Context("content hash determinism", func() {
		It("produces the same hash for the same input", func() {
			raw := map[string]string{"NoticeId": "ABC", "Title": "Hello"}
			row1 := ingest.ExtractSnapCSVRow(raw)
			row2 := ingest.ExtractSnapCSVRow(raw)
			Expect(row1.ContentHash).To(Equal(row2.ContentHash))
		})

		It("produces different hashes for different input", func() {
			raw1 := map[string]string{"NoticeId": "ABC", "Title": "Hello"}
			raw2 := map[string]string{"NoticeId": "ABC", "Title": "World"}
			row1 := ingest.ExtractSnapCSVRow(raw1)
			row2 := ingest.ExtractSnapCSVRow(raw2)
			Expect(row1.ContentHash).NotTo(Equal(row2.ContentHash))
		})
	})
})
