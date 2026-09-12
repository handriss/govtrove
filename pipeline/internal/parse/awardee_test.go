package parse_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/handriss/govtrove/pipeline/internal/parse"
)

var _ = Describe("AwardeeName", func() {
	DescribeTable("cuts the name off the concatenated CSV blob",
		func(blob, want string) {
			Expect(parse.AwardeeName(blob)).To(Equal(want))
		},
		Entry("name, street, city, state, zip, country",
			"MCGRAW HILL LLC 1325 AVENUE OF THE AMERICAS 5TH FLO NEW YORK NY 10019 USA",
			"MCGRAW HILL LLC"),
		Entry("no street, just city and state",
			"M-80 Systems, Inc. Santa Rita GU USA", "M-80 Systems, Inc."),
		Entry("dotted suffix", "DATEX-OHMEDA, INC. Madison WI 53718 USA", "DATEX-OHMEDA, INC."),
		Entry("periods inside the suffix",
			"METALMASTERS AUTOMATED TARGET SYSTEMS, L.L.C. OKEMOS 48864-3552 USA",
			"METALMASTERS AUTOMATED TARGET SYSTEMS, L.L.C."),
		Entry("foreign suffix", "PROESL BAU GMBH Eschenbach 92676 DEU", "PROESL BAU GMBH"),
		Entry("address only, no country", "GOVAGE, INC. 12 KENWOOD DR Hazelwood MO 63042 USA", "GOVAGE, INC."),
		Entry("name with no trailing address", "THE BOEING COMPANY", "THE BOEING COMPANY"),
		Entry("comma directly after the suffix",
			"PREMIER MEDICAL DISTRIBUTORS, LLC Myrtle Beach SC 29588 USA",
			"PREMIER MEDICAL DISTRIBUTORS, LLC"),
	)

	DescribeTable("keeps stacked legal suffixes together",
		func(blob, want string) {
			Expect(parse.AwardeeName(blob)).To(Equal(want))
		},
		Entry("CO INC", "SLOAN IMPLEMENT COMPANY INC Assumption IL 62510 USA", "SLOAN IMPLEMENT COMPANY INC"),
		Entry("bare CO then INC", "KONZEL CONSTRUCTION CO INC Erie PA 16505 USA", "KONZEL CONSTRUCTION CO INC"),
		Entry("CO. LTD", "MACTAGGART, SCOTT & CO. LTD", "MACTAGGART, SCOTT & CO. LTD"),
		Entry("comma joins the suffixes with no space",
			"KOREA CONTAINER POOL CO.,LTD 63-8, MAPO-DAERO, MAPO-GU SEOUL 04157",
			"KOREA CONTAINER POOL CO.,LTD"),
		Entry("mixed case, comma-joined",
			"YEAJIN E&C CO.,Ltd. 1F, 271, APSANSUNHWAN-RO, DALSEO-GU DAEGU 42819",
			"YEAJIN E&C CO.,Ltd."),
		Entry("spelled out limited liability company",
			"Fragola-G Limited Liability Company Reston VA USA",
			"Fragola-G Limited Liability Company"),
	)

	Describe("ambiguity guards", func() {
		It("does not treat the Colorado state code as a company suffix", func() {
			// "CO" here is the state in "Denver CO 80202", not a legal suffix. Cutting
			// there would have produced "...Employment Denver CO".
			Expect(parse.AwardeeName("Colorado Department of Labor and Employment Denver CO 80202 USA")).
				To(Equal(""))
		})

		It("does not match a suffix inside an initialism", func() {
			// "S.A" sits inside "U.S.A." — a substring match would cut after "U.S.A."
			Expect(parse.AwardeeName("U.S.A. SPARES INC. Miami FL 33172 USA")).
				To(Equal("U.S.A. SPARES INC."))
		})

		It("returns empty when no legal suffix is present", func() {
			Expect(parse.AwardeeName("OREGON STATE UNIVERSITY Corvallis OR 97331 USA")).To(Equal(""))
			Expect(parse.AwardeeName("Texas A&M AgriLife Research College Station TX 77843 USA")).To(Equal(""))
		})

		It("handles empty and whitespace input", func() {
			Expect(parse.AwardeeName("")).To(Equal(""))
			Expect(parse.AwardeeName("   ")).To(Equal(""))
		})

		It("never returns more than the input", func() {
			// The parser may stop short of the full legal name, but must never absorb
			// address text — a name carrying a street or city silently splits a company.
			blob := "LABORATORY CORPORATION OF AMERICA HOLDINGS Burlington NC 27215 USA"
			Expect(parse.AwardeeName(blob)).To(Equal("LABORATORY CORPORATION"))
		})
	})
})

var _ = Describe("AwardeeNameOrBlob", func() {
	It("prefers a clean API name verbatim", func() {
		Expect(parse.AwardeeNameOrBlob("Amentum Technology, Inc.",
			"Amentum Technology, Inc. 604 WILLIAM NORTHERN BLVD. Tullahoma TN 37388 USA")).
			To(Equal("Amentum Technology, Inc."))
	})

	It("parses an API name that carries the address too", func() {
		// About 9% of API names come back concatenated, same as the CSV column.
		Expect(parse.AwardeeNameOrBlob(
			"ATVARIS LLC 880 HARRISON ST SE STE 19 LEESBURG VA USA 20175-4527 ",
			"ATVARIS LLC 880 HARRISON ST SE STE 19 LEESBURG VA USA 20175-4527")).
			To(Equal("ATVARIS LLC"))
	})

	It("falls back to the CSV blob when the API gave nothing", func() {
		Expect(parse.AwardeeNameOrBlob("", "WEBB-STILES COMPANY Valley City OH 44280 USA")).
			To(Equal("WEBB-STILES COMPANY"))
	})

	It("keeps a clean API name containing digits", func() {
		// "3DB LABS INC" and "M-80 Systems" have digits but no address.
		Expect(parse.AwardeeNameOrBlob("3DB LABS INC", "3DB LABS INC West Chester OH 45069 USA")).
			To(Equal("3DB LABS INC"))
	})

	It("returns empty when neither side yields a name", func() {
		Expect(parse.AwardeeNameOrBlob("", "")).To(Equal(""))
	})

	It("strips an address that starts with a single-digit street number", func() {
		blob := "MACROSOFT INC 2 SYLVAN WAY PARSIPPANY NJ USA 07054-3809"
		Expect(parse.AwardeeNameOrBlob(blob, blob)).To(Equal("MACROSOFT INC"))
	})

	It("strips an address whose street number is followed by an ordinal", func() {
		blob := "HCA ASSET MANAGEMENT, LLC 5214 4TH AVENUE CIR E BRADENTON FL USA 34208-5621"
		Expect(parse.AwardeeNameOrBlob(blob, blob)).To(Equal("HCA ASSET MANAGEMENT, LLC"))
	})

	It("strips an address that opens with a non-numeric street token", func() {
		blob := "MULTI AIR SERVICES ENGINEERS, CORP. CARR.955 KM 29.3 RIO GRANDE PR USA 00745"
		Expect(parse.AwardeeNameOrBlob(blob, blob)).To(Equal("MULTI AIR SERVICES ENGINEERS, CORP."))
	})

	It("keeps a name whose own tail follows the legal suffix", func() {
		// "INC" leads the name here; cutting at it would leave "INC" alone. The
		// tail has no digits, so this is a name, not a name plus an address.
		Expect(parse.AwardeeNameOrBlob("INC RESEARCH LLC", "")).To(Equal("INC RESEARCH LLC"))
	})

	It("cuts correctly when the blob has leading whitespace", func() {
		// Token offsets index the trimmed string; slicing the raw blob shifted them.
		Expect(parse.AwardeeName("   ATVARIS LLC 880 HARRISON ST LEESBURG VA USA 20175")).
			To(Equal("ATVARIS LLC"))
	})

	It("leaves the field empty when an addressed name has no legal suffix", func() {
		// Nothing to cut at, so storing the API value whole would put the street
		// address in the name. Empty is the documented preference over a guess.
		blob := "ASPEN ENVIRONMENTAL GROUP 5020 CHESEBRO RD AGOURA HILLS CA USA 91301-4316"
		Expect(parse.AwardeeNameOrBlob(blob, blob)).To(Equal(""))
	})

	It("keeps a suffix-less name that ends in the country", func() {
		// "USA" alone must not read as an address, or this name is discarded.
		Expect(parse.AwardeeNameOrBlob("SIEMENS USA", "")).To(Equal("SIEMENS USA"))
	})
})
