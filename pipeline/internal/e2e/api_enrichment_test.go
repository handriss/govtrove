package e2e_test

import (
	"context"
	"encoding/json"
	"time"

	"github.com/google/uuid"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/handriss/govtrove/pipeline/internal/reconcile"
	"github.com/handriss/govtrove/pipeline/internal/samgov"
)

// CSV fixture for API enrichment tests.
// API-ENR-001: has awardee with address, PoP names, no codes.
// API-ENR-002 is API-only (not in this CSV).
var apiEnrichmentCSV = csvHeaders + "\n" +
	row(map[int]string{
		0: "API-ENR-001", 1: "SOL-ENR-001", 2: "API Enrichment Test", 3: "Solicitation", 4: "Solicitation",
		5: "01/20/2026", 6: "03/15/2026", 13: "Yes",
		14: "Department of Defense", 15: "Army", 17: "021",
		22: "100000", 23: "ACME CORP Arlington VA 22201 US",
		37: "123 Main St", 38: "Arlington", 39: "VA", 40: "22201", 41: "US",
	})

// Archived CSV used only for the resolve step (contains nothing relevant here).
var apiEnrichmentArchivedCSV = csvHeaders + "\n" +
	row(map[int]string{
		0: "API-ENR-ARCHIVED", 2: "Archived Placeholder", 3: "Solicitation", 13: "No",
	})

func apiUpsert(ctx context.Context, data []samgov.OpportunityData, snapshotDate time.Time) error {
	runID := uuid.New()
	opps := make([]reconcile.Opportunity, len(data))
	for i, d := range data {
		opps[i] = reconcile.FromAPI(d)
	}
	_, err := db.UpsertOpportunitiesFromAPI(ctx, runID, snapshotDate, opps)
	return err
}

func queryResourceLinks(ctx context.Context, noticeID string) []string {
	var raw *string
	err := db.Pool().QueryRow(ctx,
		"SELECT resource_links::text FROM opportunities WHERE notice_id = $1 AND is_latest = true",
		noticeID,
	).Scan(&raw)
	if err != nil || raw == nil {
		return nil
	}
	var links []string
	json.Unmarshal([]byte(*raw), &links)
	return links
}

var _ = Describe("API Enrichment E2E", Ordered, func() {
	var (
		ctx          context.Context
		csvSnapshot  time.Time
		apiSnapshot1 time.Time
		apiSnapshot2 time.Time
	)

	BeforeAll(func() {
		ctx = context.Background()
		csvSnapshot = time.Date(2026, 3, 1, 12, 0, 0, 0, time.UTC)
		apiSnapshot1 = time.Date(2026, 3, 1, 14, 0, 0, 0, time.UTC)
		apiSnapshot2 = time.Date(2026, 3, 2, 14, 0, 0, 0, time.UTC)
	})

	// ================================================================
	// Phase 1: CSV baseline
	// ================================================================
	Context("Phase 1: CSV baseline for API-ENR-001", func() {
		It("ingests and reconciles CSV data", func() {
			activeRunID, err := simulateIngest(ctx, db, apiEnrichmentCSV, "snapshot-csv", csvSnapshot)
			Expect(err).NotTo(HaveOccurred())

			archivedRunID, err := simulateIngest(ctx, db, apiEnrichmentArchivedCSV, "ingest-archived", csvSnapshot)
			Expect(err).NotTo(HaveOccurred())

			err = simulateReconcile(ctx, db, activeRunID, archivedRunID, csvSnapshot)
			Expect(err).NotTo(HaveOccurred())
		})

		It("has CSV awardee but no PoP codes, no awardee_name, no resource_links", func() {
			var awardee, awardeeName, popCity, popCityCode, popStateCode, popCountryCode *string
			err := db.Pool().QueryRow(ctx,
				`SELECT awardee, awardee_name, pop_city, pop_city_code, pop_state_code, pop_country_code
				 FROM opportunities WHERE notice_id = 'API-ENR-001' AND is_latest = true`,
			).Scan(&awardee, &awardeeName, &popCity, &popCityCode, &popStateCode, &popCountryCode)
			Expect(err).NotTo(HaveOccurred())

			Expect(awardee).NotTo(BeNil())
			Expect(*awardee).To(Equal("ACME CORP Arlington VA 22201 US"))
			Expect(awardeeName).To(BeNil())
			Expect(popCity).NotTo(BeNil())
			Expect(*popCity).To(Equal("Arlington"))
			Expect(popCityCode).To(BeNil())
			Expect(popStateCode).To(BeNil())
			Expect(popCountryCode).To(BeNil())

			links := queryResourceLinks(ctx, "API-ENR-001")
			Expect(links).To(BeEmpty())
		})
	})

	// ================================================================
	// Phase 2: API enrichment adds codes, awardee_name, resource_links
	// ================================================================
	Context("Phase 2: API enrichment of existing CSV record", func() {
		It("upserts API data with PoP codes, awardee_name, and resource_links", func() {
			err := apiUpsert(ctx, []samgov.OpportunityData{
				{
					NoticeID:           "API-ENR-001",
					SolicitationNumber: "SOL-ENR-001",
					Title:              "API Enrichment Test",
					Type:               "Solicitation",
					BaseType:           "Solicitation",
					PostedDate:         "01/20/2026",
					ResponseDeadLine:   "03/15/2026",
					Active:             "Yes",
					FullParentPathName: "Department of Defense.Army",
					FullParentPathCode: "021.W91CRB",
					PlaceOfPerformance: &samgov.PoP{
						StreetAddress: "123 Main St",
						City:          &samgov.NameCode{Code: "51013", Name: "Arlington"},
						State:         &samgov.NameCode{Code: "VA", Name: "Virginia"},
						Country:       &samgov.NameCode{Code: "USA", Name: "United States"},
						Zip:           "22201",
					},
					Award: &samgov.Award{
						Amount:  "100000",
						Awardee: samgov.Awardee{Name: "ACME CORP"},
					},
					ResourceLinks: []string{
						"https://sam.gov/api/prod/opps/v3/opportunities/resources/files/abc123/download",
						"https://sam.gov/api/prod/opps/v3/opportunities/resources/files/def456/download",
					},
				},
			}, apiSnapshot1)
			Expect(err).NotTo(HaveOccurred())
		})

		It("populates pop_city_code, pop_state_code, pop_country_code", func() {
			var popCityCode, popStateCode, popCountryCode *string
			err := db.Pool().QueryRow(ctx,
				`SELECT pop_city_code, pop_state_code, pop_country_code
				 FROM opportunities WHERE notice_id = 'API-ENR-001' AND is_latest = true`,
			).Scan(&popCityCode, &popStateCode, &popCountryCode)
			Expect(err).NotTo(HaveOccurred())

			Expect(popCityCode).NotTo(BeNil())
			Expect(*popCityCode).To(Equal("51013"))
			Expect(popStateCode).NotTo(BeNil())
			Expect(*popStateCode).To(Equal("VA"))
			Expect(popCountryCode).NotTo(BeNil())
			Expect(*popCountryCode).To(Equal("USA"))
		})

		It("preserves the richer CSV awardee and sets awardee_name from API", func() {
			var awardee, awardeeName *string
			err := db.Pool().QueryRow(ctx,
				`SELECT awardee, awardee_name
				 FROM opportunities WHERE notice_id = 'API-ENR-001' AND is_latest = true`,
			).Scan(&awardee, &awardeeName)
			Expect(err).NotTo(HaveOccurred())

			// CSV awardee has address — must not be overwritten by shorter API value
			Expect(awardee).NotTo(BeNil())
			Expect(*awardee).To(Equal("ACME CORP Arlington VA 22201 US"))

			// awardee_name populated from API
			Expect(awardeeName).NotTo(BeNil())
			Expect(*awardeeName).To(Equal("ACME CORP"))
		})

		It("stores resource_links as JSONB array", func() {
			links := queryResourceLinks(ctx, "API-ENR-001")
			Expect(links).To(HaveLen(2))
			Expect(links).To(ContainElement("https://sam.gov/api/prod/opps/v3/opportunities/resources/files/abc123/download"))
			Expect(links).To(ContainElement("https://sam.gov/api/prod/opps/v3/opportunities/resources/files/def456/download"))
		})

		It("marks data_sources as csv+api", func() {
			var dataSources *string
			err := db.Pool().QueryRow(ctx,
				"SELECT data_sources FROM opportunities WHERE notice_id = 'API-ENR-001' AND is_latest = true",
			).Scan(&dataSources)
			Expect(err).NotTo(HaveOccurred())
			Expect(dataSources).NotTo(BeNil())
			Expect(*dataSources).To(Equal("csv+api"))
		})

		It("preserves existing PoP name fields alongside new codes", func() {
			var popCity, popState, popCountry, popZip *string
			err := db.Pool().QueryRow(ctx,
				`SELECT pop_city, pop_state, pop_country, pop_zip
				 FROM opportunities WHERE notice_id = 'API-ENR-001' AND is_latest = true`,
			).Scan(&popCity, &popState, &popCountry, &popZip)
			Expect(err).NotTo(HaveOccurred())

			// API provides slightly different names (e.g. "Virginia" vs CSV "VA") — API wins via coalesce
			Expect(popCity).NotTo(BeNil())
			Expect(popState).NotTo(BeNil())
			Expect(popCountry).NotTo(BeNil())
			Expect(popZip).NotTo(BeNil())
			Expect(*popZip).To(Equal("22201"))
		})
	})

	// ================================================================
	// Phase 3: Changed resource_links trigger versioning
	// ================================================================
	Context("Phase 3: resource link changes create new version", func() {
		It("upserts API data with different resource_links", func() {
			err := apiUpsert(ctx, []samgov.OpportunityData{
				{
					NoticeID:           "API-ENR-001",
					SolicitationNumber: "SOL-ENR-001",
					Title:              "API Enrichment Test",
					Type:               "Solicitation",
					BaseType:           "Solicitation",
					PostedDate:         "01/20/2026",
					ResponseDeadLine:   "03/15/2026",
					Active:             "Yes",
					FullParentPathName: "Department of Defense.Army",
					FullParentPathCode: "021.W91CRB",
					PlaceOfPerformance: &samgov.PoP{
						StreetAddress: "123 Main St",
						City:          &samgov.NameCode{Code: "51013", Name: "Arlington"},
						State:         &samgov.NameCode{Code: "VA", Name: "Virginia"},
						Country:       &samgov.NameCode{Code: "USA", Name: "United States"},
						Zip:           "22201",
					},
					Award: &samgov.Award{
						Amount:  "100000",
						Awardee: samgov.Awardee{Name: "ACME CORP"},
					},
					// Changed: removed def456, added ghi789
					ResourceLinks: []string{
						"https://sam.gov/api/prod/opps/v3/opportunities/resources/files/abc123/download",
						"https://sam.gov/api/prod/opps/v3/opportunities/resources/files/ghi789/download",
					},
				},
			}, apiSnapshot2)
			Expect(err).NotTo(HaveOccurred())
		})

		It("creates a new version for the changed record", func() {
			var version int
			err := db.Pool().QueryRow(ctx,
				"SELECT version FROM opportunities WHERE notice_id = 'API-ENR-001' AND is_latest = true",
			).Scan(&version)
			Expect(err).NotTo(HaveOccurred())
			Expect(version).To(BeNumerically(">=", 2))
		})

		It("latest version has the new resource_links", func() {
			links := queryResourceLinks(ctx, "API-ENR-001")
			Expect(links).To(HaveLen(2))
			Expect(links).To(ContainElement("https://sam.gov/api/prod/opps/v3/opportunities/resources/files/abc123/download"))
			Expect(links).To(ContainElement("https://sam.gov/api/prod/opps/v3/opportunities/resources/files/ghi789/download"))
			Expect(links).NotTo(ContainElement("https://sam.gov/api/prod/opps/v3/opportunities/resources/files/def456/download"))
		})

		It("previous version is preserved with is_latest=false", func() {
			count := queryCount(ctx,
				"SELECT COUNT(*) FROM opportunities WHERE notice_id = 'API-ENR-001' AND is_latest = false")
			Expect(count).To(BeNumerically(">=", 1))
		})

		It("still has csv+api data_sources after versioning", func() {
			var dataSources *string
			err := db.Pool().QueryRow(ctx,
				"SELECT data_sources FROM opportunities WHERE notice_id = 'API-ENR-001' AND is_latest = true",
			).Scan(&dataSources)
			Expect(err).NotTo(HaveOccurred())
			Expect(dataSources).NotTo(BeNil())
			Expect(*dataSources).To(Equal("csv+api"))
		})

		It("preserves awardee fields through versioning", func() {
			var awardee, awardeeName *string
			err := db.Pool().QueryRow(ctx,
				`SELECT awardee, awardee_name
				 FROM opportunities WHERE notice_id = 'API-ENR-001' AND is_latest = true`,
			).Scan(&awardee, &awardeeName)
			Expect(err).NotTo(HaveOccurred())
			Expect(awardee).NotTo(BeNil())
			Expect(*awardee).To(Equal("ACME CORP Arlington VA 22201 US"))
			Expect(awardeeName).NotTo(BeNil())
			Expect(*awardeeName).To(Equal("ACME CORP"))
		})
	})

	// ================================================================
	// Phase 4: API-only record (no prior CSV)
	// ================================================================
	Context("Phase 4: API-only record without CSV baseline", func() {
		It("creates a new record from API alone", func() {
			err := apiUpsert(ctx, []samgov.OpportunityData{
				{
					NoticeID:           "API-ENR-002",
					SolicitationNumber: "SOL-ENR-002",
					Title:              "API-Only Opportunity",
					Type:               "Solicitation",
					BaseType:           "Solicitation",
					PostedDate:         "02/01/2026",
					Active:             "Yes",
					FullParentPathName: "NASA",
					FullParentPathCode: "080",
					PlaceOfPerformance: &samgov.PoP{
						City:    &samgov.NameCode{Code: "06075", Name: "San Francisco"},
						State:   &samgov.NameCode{Code: "CA", Name: "California"},
						Country: &samgov.NameCode{Code: "USA", Name: "United States"},
					},
					Award: &samgov.Award{
						Number:  "NNX26-001",
						Amount:  "250000",
						Awardee: samgov.Awardee{Name: "TECHCO LLC"},
					},
					ResourceLinks: []string{"https://sam.gov/api/prod/opps/v3/opportunities/resources/files/xyz999/download"},
				},
			}, apiSnapshot1)
			Expect(err).NotTo(HaveOccurred())
		})

		It("has PoP codes populated", func() {
			var popCityCode, popStateCode, popCountryCode *string
			err := db.Pool().QueryRow(ctx,
				`SELECT pop_city_code, pop_state_code, pop_country_code
				 FROM opportunities WHERE notice_id = 'API-ENR-002' AND is_latest = true`,
			).Scan(&popCityCode, &popStateCode, &popCountryCode)
			Expect(err).NotTo(HaveOccurred())
			Expect(popCityCode).NotTo(BeNil())
			Expect(*popCityCode).To(Equal("06075"))
			Expect(popStateCode).NotTo(BeNil())
			Expect(*popStateCode).To(Equal("CA"))
			Expect(popCountryCode).NotTo(BeNil())
			Expect(*popCountryCode).To(Equal("USA"))
		})

		It("has awardee_name set (no awardee since no CSV)", func() {
			var awardee, awardeeName *string
			err := db.Pool().QueryRow(ctx,
				`SELECT awardee, awardee_name
				 FROM opportunities WHERE notice_id = 'API-ENR-002' AND is_latest = true`,
			).Scan(&awardee, &awardeeName)
			Expect(err).NotTo(HaveOccurred())

			// API sets AwardeeName, not Awardee
			Expect(awardee).To(BeNil())
			Expect(awardeeName).NotTo(BeNil())
			Expect(*awardeeName).To(Equal("TECHCO LLC"))
		})

		It("has resource_links populated", func() {
			links := queryResourceLinks(ctx, "API-ENR-002")
			Expect(links).To(HaveLen(1))
			Expect(links[0]).To(Equal("https://sam.gov/api/prod/opps/v3/opportunities/resources/files/xyz999/download"))
		})

		It("has data_sources = api", func() {
			var dataSources *string
			err := db.Pool().QueryRow(ctx,
				"SELECT data_sources FROM opportunities WHERE notice_id = 'API-ENR-002' AND is_latest = true",
			).Scan(&dataSources)
			Expect(err).NotTo(HaveOccurred())
			Expect(dataSources).NotTo(BeNil())
			Expect(*dataSources).To(Equal("api"))
		})

		It("is version 1", func() {
			var version int
			err := db.Pool().QueryRow(ctx,
				"SELECT version FROM opportunities WHERE notice_id = 'API-ENR-002' AND is_latest = true",
			).Scan(&version)
			Expect(err).NotTo(HaveOccurred())
			Expect(version).To(Equal(1))
		})
	})

	// ================================================================
	// Phase 5: Nil / partial PoP handling
	// ================================================================
	Context("Phase 5: nil and partial PlaceOfPerformance", func() {
		It("handles nil PlaceOfPerformance — all PoP fields empty", func() {
			err := apiUpsert(ctx, []samgov.OpportunityData{
				{
					NoticeID:           "API-ENR-003",
					Title:              "No PoP Record",
					Type:               "Solicitation",
					Active:             "Yes",
					PlaceOfPerformance: nil,
				},
			}, apiSnapshot1)
			Expect(err).NotTo(HaveOccurred())

			var popCity, popCityCode, popStateCode, popCountryCode *string
			err = db.Pool().QueryRow(ctx,
				`SELECT pop_city, pop_city_code, pop_state_code, pop_country_code
				 FROM opportunities WHERE notice_id = 'API-ENR-003' AND is_latest = true`,
			).Scan(&popCity, &popCityCode, &popStateCode, &popCountryCode)
			Expect(err).NotTo(HaveOccurred())
			Expect(popCity).To(BeNil())
			Expect(popCityCode).To(BeNil())
			Expect(popStateCode).To(BeNil())
			Expect(popCountryCode).To(BeNil())
		})

		It("handles partial PoP — state code but no city code", func() {
			err := apiUpsert(ctx, []samgov.OpportunityData{
				{
					NoticeID: "API-ENR-004",
					Title:    "Partial PoP Record",
					Type:     "Solicitation",
					Active:   "Yes",
					PlaceOfPerformance: &samgov.PoP{
						City:    nil,
						State:   &samgov.NameCode{Code: "TX", Name: "Texas"},
						Country: &samgov.NameCode{Code: "USA", Name: "United States"},
					},
				},
			}, apiSnapshot1)
			Expect(err).NotTo(HaveOccurred())

			var popCity, popCityCode, popStateCode, popCountryCode *string
			err = db.Pool().QueryRow(ctx,
				`SELECT pop_city, pop_city_code, pop_state_code, pop_country_code
				 FROM opportunities WHERE notice_id = 'API-ENR-004' AND is_latest = true`,
			).Scan(&popCity, &popCityCode, &popStateCode, &popCountryCode)
			Expect(err).NotTo(HaveOccurred())
			Expect(popCity).To(BeNil())
			Expect(popCityCode).To(BeNil())
			Expect(popStateCode).NotTo(BeNil())
			Expect(*popStateCode).To(Equal("TX"))
			Expect(popCountryCode).NotTo(BeNil())
			Expect(*popCountryCode).To(Equal("USA"))
		})
	})

	// ================================================================
	// Phase 6: Idempotent API re-ingestion (same data, no new version)
	// ================================================================
	Context("Phase 6: idempotent re-ingestion does not create spurious versions", func() {
		It("re-upserts API-ENR-002 with identical data", func() {
			err := apiUpsert(ctx, []samgov.OpportunityData{
				{
					NoticeID:           "API-ENR-002",
					SolicitationNumber: "SOL-ENR-002",
					Title:              "API-Only Opportunity",
					Type:               "Solicitation",
					BaseType:           "Solicitation",
					PostedDate:         "02/01/2026",
					Active:             "Yes",
					FullParentPathName: "NASA",
					FullParentPathCode: "080",
					PlaceOfPerformance: &samgov.PoP{
						City:    &samgov.NameCode{Code: "06075", Name: "San Francisco"},
						State:   &samgov.NameCode{Code: "CA", Name: "California"},
						Country: &samgov.NameCode{Code: "USA", Name: "United States"},
					},
					Award: &samgov.Award{
						Number:  "NNX26-001",
						Amount:  "250000",
						Awardee: samgov.Awardee{Name: "TECHCO LLC"},
					},
					ResourceLinks: []string{"https://sam.gov/api/prod/opps/v3/opportunities/resources/files/xyz999/download"},
				},
			}, apiSnapshot2)
			Expect(err).NotTo(HaveOccurred())
		})

		It("does not create a new version", func() {
			var version int
			err := db.Pool().QueryRow(ctx,
				"SELECT version FROM opportunities WHERE notice_id = 'API-ENR-002' AND is_latest = true",
			).Scan(&version)
			Expect(err).NotTo(HaveOccurred())
			Expect(version).To(Equal(1))

			totalRows := queryCount(ctx,
				"SELECT COUNT(*) FROM opportunities WHERE notice_id = 'API-ENR-002'")
			Expect(totalRows).To(Equal(1))
		})
	})

	// ================================================================
	// Phase 7: Empty resource links from API don't clear existing
	// ================================================================
	Context("Phase 7: empty API resource_links preserve existing", func() {
		It("upserts API-ENR-002 with empty resource_links", func() {
			err := apiUpsert(ctx, []samgov.OpportunityData{
				{
					NoticeID:           "API-ENR-002",
					SolicitationNumber: "SOL-ENR-002",
					Title:              "API-Only Opportunity",
					Type:               "Solicitation",
					BaseType:           "Solicitation",
					PostedDate:         "02/01/2026",
					Active:             "Yes",
					FullParentPathName: "NASA",
					FullParentPathCode: "080",
					PlaceOfPerformance: &samgov.PoP{
						City:    &samgov.NameCode{Code: "06075", Name: "San Francisco"},
						State:   &samgov.NameCode{Code: "CA", Name: "California"},
						Country: &samgov.NameCode{Code: "USA", Name: "United States"},
					},
					Award: &samgov.Award{
						Number:  "NNX26-001",
						Amount:  "250000",
						Awardee: samgov.Awardee{Name: "TECHCO LLC"},
					},
					ResourceLinks: nil,
				},
			}, apiSnapshot2)
			Expect(err).NotTo(HaveOccurred())
		})

		It("still has resource_links from previous ingestion", func() {
			links := queryResourceLinks(ctx, "API-ENR-002")
			Expect(links).To(HaveLen(1))
			Expect(links[0]).To(Equal("https://sam.gov/api/prod/opps/v3/opportunities/resources/files/xyz999/download"))
		})
	})
})
