package parse_test

import (
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/handriss/govtrove/pipeline/internal/parse"
)

var _ = Describe("Date", func() {
	DescribeTable("parses valid date formats",
		func(input string, year int, month time.Month, day int) {
			result := parse.Date(input)
			Expect(result).NotTo(BeNil())
			Expect(result.Year()).To(Equal(year))
			Expect(result.Month()).To(Equal(month))
			Expect(result.Day()).To(Equal(day))
		},
		Entry("ISO date", "2026-02-15", 2026, time.February, 15),
		Entry("ISO datetime", "2026-02-15T10:30:00", 2026, time.February, 15),
		Entry("ISO datetime with Z", "2026-02-15T10:30:00Z", 2026, time.February, 15),
		Entry("datetime with offset (short)", "2024-03-15 10:30:00-04", 2024, time.March, 15),
		Entry("datetime with offset (long)", "2024-03-15 10:30:00-0400", 2024, time.March, 15),
		Entry("datetime with millis", "2024-03-15 10:30:00.123-04", 2024, time.March, 15),
		Entry("US date format", "03/15/2024", 2024, time.March, 15),
		Entry("US date with time", "03/15/2024 10:30", 2024, time.March, 15),
		Entry("space-separated datetime", "2024-03-15 10:30:00", 2024, time.March, 15),
	)

	It("returns nil for empty input", func() {
		Expect(parse.Date("")).To(BeNil())
	})

	It("returns nil for garbage input", func() {
		Expect(parse.Date("not-a-date")).To(BeNil())
	})
})

var _ = Describe("DateOnly", func() {
	DescribeTable("truncates to midnight UTC",
		func(input string, year int, month time.Month, day int) {
			result := parse.DateOnly(input)
			Expect(result).NotTo(BeNil())

			want := time.Date(year, month, day, 0, 0, 0, 0, time.UTC)
			Expect(result.Equal(want)).To(BeTrue(), "expected %v, got %v", want, result)
		},
		Entry("ISO date", "2026-02-15", 2026, time.February, 15),
		Entry("midnight timestamp", "2021-02-18T00:00:00", 2021, time.February, 18),
		Entry("timestamp with offset", "2024-03-15 10:30:00-04", 2024, time.March, 15),
		Entry("sentinel date (still parses)", "1969-12-31", 1969, time.December, 31),
	)

	It("returns nil for empty input", func() {
		Expect(parse.DateOnly("")).To(BeNil())
	})

	It("returns nil for garbage input", func() {
		Expect(parse.DateOnly("garbage")).To(BeNil())
	})
})

var _ = Describe("IsSentinelDate", func() {
	DescribeTable("detects pre-1980 dates as sentinels",
		func(t *time.Time, expected bool) {
			Expect(parse.IsSentinelDate(t)).To(Equal(expected))
		},
		Entry("nil", nil, false),
		Entry("unix epoch sentinel", timePtr(1969, time.December, 31), true),
		Entry("epoch start", timePtr(1970, time.January, 1), true),
		Entry("boundary (1980-01-01)", timePtr(1980, time.January, 1), false),
		Entry("modern date", timePtr(2026, time.February, 15), false),
	)
})

var _ = Describe("Active", func() {
	DescribeTable("parses boolean-like strings",
		func(input string, expected bool) {
			Expect(parse.Active(input)).To(Equal(expected))
		},
		Entry("yes", "Yes", true),
		Entry("true", "true", true),
		Entry("1", "1", true),
		Entry("no", "No", false),
		Entry("false", "false", false),
		Entry("0", "0", false),
		Entry("empty", "", false),
		Entry("with whitespace", "  yes  ", true),
	)
})

var _ = Describe("Amount", func() {
	DescribeTable("parses dollar amounts",
		func(input string, expected float64) {
			result := parse.Amount(input)
			Expect(result).NotTo(BeNil())
			Expect(*result).To(BeNumerically("~", expected, 0.01))
		},
		Entry("plain number", "1234.56", 1234.56),
		Entry("with commas", "1,234,567.89", 1234567.89),
		Entry("with dollar sign", "$1,234.56", 1234.56),
		Entry("integer", "500", 500.0),
	)

	It("returns nil for empty input", func() {
		Expect(parse.Amount("")).To(BeNil())
	})

	It("returns nil for non-numeric input", func() {
		Expect(parse.Amount("not-a-number")).To(BeNil())
	})
})

func timePtr(year int, month time.Month, day int) *time.Time {
	t := time.Date(year, month, day, 0, 0, 0, 0, time.UTC)
	return &t
}
