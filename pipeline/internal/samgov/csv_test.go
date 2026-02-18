package samgov_test

import (
	"log/slog"
	"strings"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/handriss/govtrove/pipeline/internal/samgov"
)

var logger = slog.Default()

var _ = Describe("ParseCSVFromReader", func() {
	Context("with a well-formed CSV", func() {
		It("parses all rows with correct field values", func() {
			csv := "NoticeId,Title,Type,Active\n" +
				"ID-001,First Opportunity,Solicitation,Yes\n" +
				"ID-002,Second Opportunity,Award,No\n"

			result, err := samgov.ParseCSVFromReader(strings.NewReader(csv), 0, logger)

			Expect(err).NotTo(HaveOccurred())
			Expect(result.ParsedRows).To(Equal(2))
			Expect(result.Errors).To(Equal(0))
			Expect(result.Headers).To(Equal([]string{"NoticeId", "Title", "Type", "Active"}))
			Expect(result.Rows).To(HaveLen(2))
			Expect(result.Rows[0]["NoticeId"]).To(Equal("ID-001"))
			Expect(result.Rows[0]["Title"]).To(Equal("First Opportunity"))
			Expect(result.Rows[1]["NoticeId"]).To(Equal("ID-002"))
		})
	})

	Context("when rows have missing NoticeId", func() {
		It("skips them and counts as errors", func() {
			csv := "NoticeId,Title\n" +
				"ID-001,Good Row\n" +
				",Missing ID\n" +
				"ID-003,Another Good Row\n"

			result, err := samgov.ParseCSVFromReader(strings.NewReader(csv), 0, logger)

			Expect(err).NotTo(HaveOccurred())
			Expect(result.ParsedRows).To(Equal(2))
			Expect(result.Errors).To(Equal(1))
			Expect(result.Rows).To(HaveLen(2))
			Expect(result.Rows[0]["NoticeId"]).To(Equal("ID-001"))
			Expect(result.Rows[1]["NoticeId"]).To(Equal("ID-003"))
		})
	})

	Context("with lazy quotes (malformed CSV)", func() {
		It("parses rows without error", func() {
			csv := "NoticeId,Title\n" +
				"ID-001,\"Some \"quoted\" value\"\n" +
				"ID-002,Normal Value\n"

			result, err := samgov.ParseCSVFromReader(strings.NewReader(csv), 0, logger)

			Expect(err).NotTo(HaveOccurred())
			Expect(result.ParsedRows).To(BeNumerically(">=", 1))
		})
	})

	Context("when a row limit is set", func() {
		It("stops at the limit", func() {
			csv := "NoticeId,Title\n" +
				"ID-001,Row 1\n" +
				"ID-002,Row 2\n" +
				"ID-003,Row 3\n" +
				"ID-004,Row 4\n" +
				"ID-005,Row 5\n"

			result, err := samgov.ParseCSVFromReader(strings.NewReader(csv), 3, logger)

			Expect(err).NotTo(HaveOccurred())
			Expect(result.ParsedRows).To(Equal(3))
			Expect(result.Rows).To(HaveLen(3))
		})
	})

	Context("with an empty CSV (headers only)", func() {
		It("returns zero rows and no error", func() {
			csv := "NoticeId,Title,Type\n"

			result, err := samgov.ParseCSVFromReader(strings.NewReader(csv), 0, logger)

			Expect(err).NotTo(HaveOccurred())
			Expect(result.ParsedRows).To(Equal(0))
			Expect(result.Errors).To(Equal(0))
			Expect(result.Rows).To(BeEmpty())
			Expect(result.Headers).To(Equal([]string{"NoticeId", "Title", "Type"}))
		})
	})

	Context("with whitespace in headers and values", func() {
		It("trims whitespace from both", func() {
			csv := " NoticeId , Title \n" +
				" ID-001 , Trimmed Value \n"

			result, err := samgov.ParseCSVFromReader(strings.NewReader(csv), 0, logger)

			Expect(err).NotTo(HaveOccurred())
			Expect(result.ParsedRows).To(Equal(1))
			Expect(result.Headers).To(ContainElement("NoticeId"))
			Expect(result.Rows[0]["NoticeId"]).To(Equal("ID-001"))
			Expect(result.Rows[0]["Title"]).To(Equal("Trimmed Value"))
		})
	})

	Context("with empty values", func() {
		It("omits empty fields from the row map", func() {
			csv := "NoticeId,Title,Description\n" +
				"ID-001,,\n"

			result, err := samgov.ParseCSVFromReader(strings.NewReader(csv), 0, logger)

			Expect(err).NotTo(HaveOccurred())
			Expect(result.ParsedRows).To(Equal(1))
			Expect(result.Rows[0]).To(HaveKey("NoticeId"))
			Expect(result.Rows[0]).NotTo(HaveKey("Title"))
			Expect(result.Rows[0]).NotTo(HaveKey("Description"))
		})
	})

	Context("with fewer columns than headers", func() {
		It("only populates available columns", func() {
			csv := "NoticeId,Title,Type\n" +
				"ID-001,Short\n"

			result, err := samgov.ParseCSVFromReader(strings.NewReader(csv), 0, logger)

			Expect(err).NotTo(HaveOccurred())
			Expect(result.ParsedRows).To(Equal(1))
			Expect(result.Rows[0]["NoticeId"]).To(Equal("ID-001"))
			Expect(result.Rows[0]["Title"]).To(Equal("Short"))
			Expect(result.Rows[0]).NotTo(HaveKey("Type"))
		})
	})

	Context("with no headers", func() {
		It("returns an error", func() {
			csv := ""

			_, err := samgov.ParseCSVFromReader(strings.NewReader(csv), 0, logger)

			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("headers"))
		})
	})
})
