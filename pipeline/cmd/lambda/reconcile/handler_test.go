package main

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/google/uuid"
	"github.com/handriss/govtrove/pipeline/internal/database"
	"github.com/handriss/govtrove/pipeline/internal/reconcile"
	"github.com/handriss/govtrove/pipeline/internal/testutil"
)

var _ = Describe("Reconcile Handler", func() {
	var (
		h     *Handler
		store *testutil.MockStore
		ctx   context.Context
	)

	BeforeEach(func() {
		ctx = context.Background()
		store = &testutil.MockStore{}
		h = &Handler{Store: store, Logger: slog.Default()}
	})

	buildEvent := func(results []IngestionResult, apiResult *IngestionResult) json.RawMessage {
		input := Input{
			ExecutionID:    uuid.New().String(),
			IngestionResults: results,
			APIResult:        apiResult,
		}
		b, _ := json.Marshal(input)
		return b
	}

	Context("with active ingestion results", func() {
		var activeRunID uuid.UUID

		BeforeEach(func() {
			activeRunID = uuid.New()

			store.GetSnapCSVRawDataFn = func(_ context.Context, runID uuid.UUID) ([]map[string]string, error) {
				return []map[string]string{
					{"NoticeId": "OPP-001", "Title": "Test", "ArchiveDate": "2026-06-01"},
					{"NoticeId": "OPP-002", "Title": "Test 2", "ArchiveDate": "2026-07-01"},
				}, nil
			}

			store.UpsertOpportunitiesFn = func(_ context.Context, _ uuid.UUID, _ time.Time, opps []reconcile.Opportunity) (int, error) {
				return len(opps), nil
			}
		})

		It("upserts opportunities and marks disappeared inactive", func() {
			event := buildEvent([]IngestionResult{
				{Status: "ok", RunID: activeRunID.String(), JobType: "snapshot-csv"},
			}, nil)

			output, err := h.Handle(ctx, event)

			Expect(err).NotTo(HaveOccurred())
			Expect(output.Status).To(Equal("ok"))
		})

		It("collects data quality issues and inserts them before upserting", func() {
			store.GetSnapCSVRawDataFn = func(_ context.Context, _ uuid.UUID) ([]map[string]string, error) {
				return []map[string]string{
					{"NoticeId": "OPP-001", "ArchiveDate": "1969-12-31"},
				}, nil
			}

			var insertedIssues []database.DataQualityEntry
			store.InsertDataQualityIssuesFn = func(_ context.Context, _ uuid.UUID, entries []database.DataQualityEntry) {
				insertedIssues = entries
			}

			event := buildEvent([]IngestionResult{
				{Status: "ok", RunID: activeRunID.String(), JobType: "snapshot-csv"},
			}, nil)

			output, err := h.Handle(ctx, event)

			Expect(err).NotTo(HaveOccurred())
			Expect(output.Status).To(Equal("ok"))
			Expect(insertedIssues).To(HaveLen(1))
			Expect(insertedIssues[0].IssueType).To(Equal("sentinel_date"))
		})

		It("calls DeactivateExpiredOpportunities", func() {
			deactivateCalled := false
			store.DeactivateExpiredOpportunitiesFn = func(_ context.Context) (int, int, error) {
				deactivateCalled = true
				return 3, 1, nil
			}

			event := buildEvent([]IngestionResult{
				{Status: "ok", RunID: activeRunID.String(), JobType: "snapshot-csv"},
			}, nil)

			output, err := h.Handle(ctx, event)

			Expect(err).NotTo(HaveOccurred())
			Expect(output.Status).To(Equal("ok"))
			Expect(deactivateCalled).To(BeTrue())
		})
	})

	Context("when an ingestion result has failed status", func() {
		It("returns an error without upserting anything", func() {
			upsertCalled := false
			store.UpsertOpportunitiesFn = func(_ context.Context, _ uuid.UUID, _ time.Time, _ []reconcile.Opportunity) (int, error) {
				upsertCalled = true
				return 0, nil
			}

			event := buildEvent([]IngestionResult{
				{Status: "failed", RunID: uuid.New().String(), JobType: "snapshot-csv"},
			}, nil)

			_, err := h.Handle(ctx, event)

			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("failed"))
			Expect(upsertCalled).To(BeFalse())
		})
	})

	Context("when active run is present", func() {
		It("calls MarkDisappearedInactive", func() {
			activeRunID := uuid.New()

			store.GetSnapCSVRawDataFn = func(_ context.Context, _ uuid.UUID) ([]map[string]string, error) {
				return []map[string]string{
					{"NoticeId": "OPP-001", "Title": "Active Only"},
				}, nil
			}

			markCalled := false
			store.MarkDisappearedInactiveFn = func(_ context.Context, _ uuid.UUID) (int, error) {
				markCalled = true
				return 0, nil
			}

			event := buildEvent([]IngestionResult{
				{Status: "ok", RunID: activeRunID.String(), JobType: "snapshot-csv"},
			}, nil)

			output, err := h.Handle(ctx, event)

			Expect(err).NotTo(HaveOccurred())
			Expect(output.Status).To(Equal("ok"))
			Expect(markCalled).To(BeTrue())
		})
	})

	Context("when GetSnapCSVRawData returns an error", func() {
		It("returns the error without partial upserts", func() {
			store.GetSnapCSVRawDataFn = func(_ context.Context, _ uuid.UUID) ([]map[string]string, error) {
				return nil, errors.New("database connection lost")
			}

			upsertCalled := false
			store.UpsertOpportunitiesFn = func(_ context.Context, _ uuid.UUID, _ time.Time, _ []reconcile.Opportunity) (int, error) {
				upsertCalled = true
				return 0, nil
			}

			event := buildEvent([]IngestionResult{
				{Status: "ok", RunID: uuid.New().String(), JobType: "snapshot-csv"},
			}, nil)

			_, err := h.Handle(ctx, event)

			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("database connection lost"))
			Expect(upsertCalled).To(BeFalse())
		})
	})

	Context("when UpsertOpportunities fails", func() {
		It("returns the error", func() {
			store.GetSnapCSVRawDataFn = func(_ context.Context, _ uuid.UUID) ([]map[string]string, error) {
				return []map[string]string{
					{"NoticeId": "OPP-001"},
				}, nil
			}
			store.UpsertOpportunitiesFn = func(_ context.Context, _ uuid.UUID, _ time.Time, _ []reconcile.Opportunity) (int, error) {
				return 0, errors.New("upsert batch failed")
			}

			event := buildEvent([]IngestionResult{
				{Status: "ok", RunID: uuid.New().String(), JobType: "snapshot-csv"},
			}, nil)

			_, err := h.Handle(ctx, event)

			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("upsert"))
		})
	})

	Context("with invalid JSON input", func() {
		It("returns an unmarshal error", func() {
			_, err := h.Handle(ctx, json.RawMessage(`{invalid`))

			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("unmarshal"))
		})
	})

	Context("with no ingestion results", func() {
		It("returns ok without doing any work", func() {
			event := buildEvent([]IngestionResult{}, nil)

			output, err := h.Handle(ctx, event)

			Expect(err).NotTo(HaveOccurred())
			Expect(output.Status).To(Equal("ok"))
		})
	})

	Context("verifies records passed to UpsertOpportunities", func() {
		It("passes all CSV notice_ids to upsert", func() {
			activeRunID := uuid.New()

			store.GetSnapCSVRawDataFn = func(_ context.Context, _ uuid.UUID) ([]map[string]string, error) {
				return []map[string]string{
					{"NoticeId": "CSV-001", "Title": "First", "Active": "Yes"},
					{"NoticeId": "CSV-002", "Title": "Second", "Active": "Yes"},
					{"NoticeId": "CSV-003", "Title": "Third", "Active": "Yes"},
				}, nil
			}

			var upsertedOpps []reconcile.Opportunity
			store.UpsertOpportunitiesFn = func(_ context.Context, _ uuid.UUID, _ time.Time, opps []reconcile.Opportunity) (int, error) {
				upsertedOpps = opps
				return len(opps), nil
			}

			event := buildEvent([]IngestionResult{
				{Status: "ok", RunID: activeRunID.String(), JobType: "snapshot-csv"},
			}, nil)

			output, err := h.Handle(ctx, event)

			Expect(err).NotTo(HaveOccurred())
			Expect(output.Status).To(Equal("ok"))
			Expect(upsertedOpps).To(HaveLen(3))
			ids := make([]string, len(upsertedOpps))
			for i, o := range upsertedOpps {
				ids[i] = o.NoticeID
			}
			Expect(ids).To(ConsistOf("CSV-001", "CSV-002", "CSV-003"))
		})

		It("uses activeRunID as upsert run_id when both CSV and API are present", func() {
			activeRunID := uuid.New()
			apiRunID := uuid.New()

			store.GetSnapCSVRawDataFn = func(_ context.Context, _ uuid.UUID) ([]map[string]string, error) {
				return []map[string]string{
					{"NoticeId": "OPP-001", "Title": "CSV", "Active": "Yes"},
				}, nil
			}
			store.GetSnapAPIRawDataFn = func(_ context.Context, _ uuid.UUID) ([]database.SnapAPIRawRow, error) {
				return []database.SnapAPIRawRow{
					{NoticeID: "API-001", RawData: json.RawMessage(`{"noticeId":"API-001","title":"API Only","active":"Yes"}`)},
				}, nil
			}

			var capturedRunID uuid.UUID
			store.UpsertOpportunitiesFn = func(_ context.Context, runID uuid.UUID, _ time.Time, opps []reconcile.Opportunity) (int, error) {
				capturedRunID = runID
				return len(opps), nil
			}

			event := buildEvent([]IngestionResult{
				{Status: "ok", RunID: activeRunID.String(), JobType: "snapshot-csv"},
			}, &IngestionResult{
				Status: "ok", RunID: apiRunID.String(), JobType: "snapshot-api",
			})

			_, err := h.Handle(ctx, event)
			Expect(err).NotTo(HaveOccurred())
			Expect(capturedRunID).To(Equal(activeRunID))
		})

		It("uses apiRunID as upsert run_id when only API is present", func() {
			apiRunID := uuid.New()

			store.GetSnapAPIRawDataFn = func(_ context.Context, _ uuid.UUID) ([]database.SnapAPIRawRow, error) {
				return []database.SnapAPIRawRow{
					{NoticeID: "API-001", RawData: json.RawMessage(`{"noticeId":"API-001","title":"API Only","active":"Yes"}`)},
				}, nil
			}

			var capturedRunID uuid.UUID
			store.UpsertOpportunitiesFn = func(_ context.Context, runID uuid.UUID, _ time.Time, opps []reconcile.Opportunity) (int, error) {
				capturedRunID = runID
				return len(opps), nil
			}

			event := buildEvent([]IngestionResult{}, &IngestionResult{
				Status: "ok", RunID: apiRunID.String(), JobType: "snapshot-api",
			})

			_, err := h.Handle(ctx, event)
			Expect(err).NotTo(HaveOccurred())
			Expect(capturedRunID).To(Equal(apiRunID))
		})
	})

	Context("CSV+API merge with overlapping and unique records", func() {
		It("merges overlapping records and includes unique from each source", func() {
			activeRunID := uuid.New()
			apiRunID := uuid.New()

			store.GetSnapCSVRawDataFn = func(_ context.Context, _ uuid.UUID) ([]map[string]string, error) {
				return []map[string]string{
					{"NoticeId": "SHARED-001", "Title": "CSV Title", "Active": "Yes"},
					{"NoticeId": "CSV-ONLY-001", "Title": "CSV Only", "Active": "Yes"},
				}, nil
			}
			store.GetSnapAPIRawDataFn = func(_ context.Context, _ uuid.UUID) ([]database.SnapAPIRawRow, error) {
				return []database.SnapAPIRawRow{
					{NoticeID: "SHARED-001", RawData: json.RawMessage(`{"noticeId":"SHARED-001","title":"API Title","active":"Yes","additionalInfoLink":"http://info"}`)},
					{NoticeID: "API-ONLY-001", RawData: json.RawMessage(`{"noticeId":"API-ONLY-001","title":"API Only","active":"Yes"}`)},
				}, nil
			}

			var upsertedOpps []reconcile.Opportunity
			store.UpsertOpportunitiesFn = func(_ context.Context, _ uuid.UUID, _ time.Time, opps []reconcile.Opportunity) (int, error) {
				upsertedOpps = opps
				return len(opps), nil
			}

			event := buildEvent([]IngestionResult{
				{Status: "ok", RunID: activeRunID.String(), JobType: "snapshot-csv"},
			}, &IngestionResult{
				Status: "ok", RunID: apiRunID.String(), JobType: "snapshot-api",
			})

			output, err := h.Handle(ctx, event)

			Expect(err).NotTo(HaveOccurred())
			Expect(output.Status).To(Equal("ok"))
			Expect(upsertedOpps).To(HaveLen(3))

			oppMap := make(map[string]reconcile.Opportunity)
			for _, o := range upsertedOpps {
				oppMap[o.NoticeID] = o
			}

			// Shared record: CSV title wins, API-only fields carried forward
			shared := oppMap["SHARED-001"]
			Expect(shared.Title).To(Equal("CSV Title"))
			Expect(shared.AdditionalInfoLink).To(Equal("http://info"))
			Expect(shared.DataSources).To(Equal("csv+api"))

			// CSV-only record
			csvOnly := oppMap["CSV-ONLY-001"]
			Expect(csvOnly.Title).To(Equal("CSV Only"))
			Expect(csvOnly.DataSources).To(Equal("csv"))

			// API-only record
			apiOnly := oppMap["API-ONLY-001"]
			Expect(apiOnly.Title).To(Equal("API Only"))
			Expect(apiOnly.DataSources).To(Equal("api"))
		})

		It("carries API-only fields into merged records", func() {
			activeRunID := uuid.New()
			apiRunID := uuid.New()

			store.GetSnapCSVRawDataFn = func(_ context.Context, _ uuid.UUID) ([]map[string]string, error) {
				return []map[string]string{
					{"NoticeId": "MERGED-001", "Title": "CSV Title", "Active": "Yes"},
				}, nil
			}
			store.GetSnapAPIRawDataFn = func(_ context.Context, _ uuid.UUID) ([]database.SnapAPIRawRow, error) {
				return []database.SnapAPIRawRow{
					{NoticeID: "MERGED-001", RawData: json.RawMessage(`{
						"noticeId":"MERGED-001",
						"title":"API Title",
						"active":"Yes",
						"fullParentPathName":"DEPT.SUBTIER.OFFICE",
						"fullParentPathCode":"001.002.003",
						"additionalInfoLink":"http://extra",
						"resourceLinks":["http://doc1.pdf","http://doc2.pdf"],
						"award":{"awardee":{"name":"Acme Corp","ueiSAM":"ABC123","location":{"streetAddress":"123 Main St","city":{"code":"NYC","name":"New York"},"state":{"code":"NY","name":"New York"},"country":{"code":"US","name":"USA"},"zip":"10001"}}}
					}`)},
				}, nil
			}

			var upsertedOpps []reconcile.Opportunity
			store.UpsertOpportunitiesFn = func(_ context.Context, _ uuid.UUID, _ time.Time, opps []reconcile.Opportunity) (int, error) {
				upsertedOpps = opps
				return len(opps), nil
			}

			event := buildEvent([]IngestionResult{
				{Status: "ok", RunID: activeRunID.String(), JobType: "snapshot-csv"},
			}, &IngestionResult{
				Status: "ok", RunID: apiRunID.String(), JobType: "snapshot-api",
			})

			_, err := h.Handle(ctx, event)
			Expect(err).NotTo(HaveOccurred())
			Expect(upsertedOpps).To(HaveLen(1))

			merged := upsertedOpps[0]
			Expect(merged.Title).To(Equal("CSV Title"))
			Expect(merged.FullParentPathName).To(Equal("DEPT.SUBTIER.OFFICE"))
			Expect(merged.FullParentPathCode).To(Equal("001.002.003"))
			Expect(merged.AdditionalInfoLink).To(Equal("http://extra"))
			Expect(merged.ResourceLinks).To(ConsistOf("http://doc1.pdf", "http://doc2.pdf"))
			Expect(merged.AwardeeName).To(Equal("Acme Corp"))
			Expect(merged.AwardeeUeiSAM).To(Equal("ABC123"))
		})
	})

	Context("pipeline step stats", func() {
		It("reports correct stats via CompletePipelineStep", func() {
			activeRunID := uuid.New()
			apiRunID := uuid.New()
			execID := uuid.New()

			store.GetSnapCSVRawDataFn = func(_ context.Context, _ uuid.UUID) ([]map[string]string, error) {
				return []map[string]string{
					{"NoticeId": "OPP-001", "Title": "A", "Active": "Yes"},
					{"NoticeId": "OPP-002", "Title": "B", "Active": "Yes"},
				}, nil
			}
			store.GetSnapAPIRawDataFn = func(_ context.Context, _ uuid.UUID) ([]database.SnapAPIRawRow, error) {
				return []database.SnapAPIRawRow{
					{NoticeID: "OPP-001", RawData: json.RawMessage(`{"noticeId":"OPP-001","title":"A-api","active":"Yes"}`)},
					{NoticeID: "API-001", RawData: json.RawMessage(`{"noticeId":"API-001","title":"API Only","active":"Yes"}`)},
				}, nil
			}

			store.UpsertOpportunitiesFn = func(_ context.Context, _ uuid.UUID, _ time.Time, opps []reconcile.Opportunity) (int, error) {
				return len(opps), nil
			}

			var capturedStats map[string]any
			store.CompletePipelineStepFn = func(_ context.Context, _ uuid.UUID, stats map[string]any, _ int) error {
				capturedStats = stats
				return nil
			}

			input := Input{
				ExecutionID: execID.String(),
				IngestionResults: []IngestionResult{
					{Status: "ok", RunID: activeRunID.String(), JobType: "snapshot-csv"},
				},
				APIResult: &IngestionResult{
					Status: "ok", RunID: apiRunID.String(), JobType: "snapshot-api",
				},
			}
			b, _ := json.Marshal(input)

			output, err := h.Handle(ctx, b)

			Expect(err).NotTo(HaveOccurred())
			Expect(output.Status).To(Equal("ok"))
			Expect(capturedStats).NotTo(BeNil())
			Expect(capturedStats["upserted"]).To(Equal(3))
			Expect(capturedStats["csv_count"]).To(Equal(2))
			Expect(capturedStats["api_count"]).To(Equal(2))
		})
	})

	Context("post-upsert housekeeping", func() {
		It("calls RefreshAgencies", func() {
			refreshCalled := false
			store.RefreshAgenciesFn = func(_ context.Context) (int, error) {
				refreshCalled = true
				return 42, nil
			}

			event := buildEvent([]IngestionResult{}, nil)
			output, err := h.Handle(ctx, event)

			Expect(err).NotTo(HaveOccurred())
			Expect(output.Status).To(Equal("ok"))
			Expect(refreshCalled).To(BeTrue())
		})

		It("calls DeleteOldSearchEvents with 90 days", func() {
			var capturedDays int
			store.DeleteOldSearchEventsFn = func(_ context.Context, days int) (int64, error) {
				capturedDays = days
				return 5, nil
			}

			event := buildEvent([]IngestionResult{}, nil)
			output, err := h.Handle(ctx, event)

			Expect(err).NotTo(HaveOccurred())
			Expect(output.Status).To(Equal("ok"))
			Expect(capturedDays).To(Equal(90))
		})
	})

	Context("with API result present", func() {
		var apiRunID uuid.UUID

		BeforeEach(func() {
			apiRunID = uuid.New()
		})

		It("loads API opps and upserts them", func() {
			store.GetSnapAPIRawDataFn = func(_ context.Context, runID uuid.UUID) ([]database.SnapAPIRawRow, error) {
				Expect(runID).To(Equal(apiRunID))
				return []database.SnapAPIRawRow{
					{
						NoticeID: "API-001",
						RawData:  json.RawMessage(`{"noticeId":"API-001","title":"API Opp","active":"Yes"}`),
					},
				}, nil
			}

			var upsertedOpps []reconcile.Opportunity
			store.UpsertOpportunitiesFn = func(_ context.Context, _ uuid.UUID, _ time.Time, opps []reconcile.Opportunity) (int, error) {
				upsertedOpps = opps
				return len(opps), nil
			}

			event := buildEvent([]IngestionResult{}, &IngestionResult{
				Status:  "ok",
				RunID:   apiRunID.String(),
				JobType: "snapshot-api",
			})

			output, err := h.Handle(ctx, event)

			Expect(err).NotTo(HaveOccurred())
			Expect(output.Status).To(Equal("ok"))
			Expect(upsertedOpps).To(HaveLen(1))
			Expect(upsertedOpps[0].NoticeID).To(Equal("API-001"))
		})

		It("returns error when API result has failed status", func() {
			event := buildEvent([]IngestionResult{}, &IngestionResult{
				Status:  "failed",
				RunID:   apiRunID.String(),
				JobType: "snapshot-api",
			})

			_, err := h.Handle(ctx, event)

			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("api ingestion failed"))
		})

		It("returns error when GetSnapAPIRawData fails", func() {
			store.GetSnapAPIRawDataFn = func(_ context.Context, _ uuid.UUID) ([]database.SnapAPIRawRow, error) {
				return nil, errors.New("snap_api query failed")
			}

			event := buildEvent([]IngestionResult{}, &IngestionResult{
				Status:  "ok",
				RunID:   apiRunID.String(),
				JobType: "snapshot-api",
			})

			_, err := h.Handle(ctx, event)

			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("snap_api"))
		})

		It("prefers CSV and logs reconcile mismatches when same notice_id differs", func() {
			activeRunID := uuid.New()

			store.GetSnapCSVRawDataFn = func(_ context.Context, _ uuid.UUID) ([]map[string]string, error) {
				return []map[string]string{
					{"NoticeId": "BOTH-001", "Title": "CSV Version", "Active": "Yes"},
				}, nil
			}

			store.GetSnapAPIRawDataFn = func(_ context.Context, _ uuid.UUID) ([]database.SnapAPIRawRow, error) {
				return []database.SnapAPIRawRow{
					{
						NoticeID: "BOTH-001",
						RawData:  json.RawMessage(`{"noticeId":"BOTH-001","title":"API Version","active":"Yes"}`),
					},
				}, nil
			}

			var upsertedOpps []reconcile.Opportunity
			store.UpsertOpportunitiesFn = func(_ context.Context, _ uuid.UUID, _ time.Time, opps []reconcile.Opportunity) (int, error) {
				upsertedOpps = opps
				return len(opps), nil
			}

			var reconcileDQ []database.ReconcileDQEntry
			store.InsertReconcileDQIssuesFn = func(_ context.Context, _, _ uuid.UUID, entries []database.ReconcileDQEntry) {
				reconcileDQ = entries
			}

			event := buildEvent([]IngestionResult{
				{Status: "ok", RunID: activeRunID.String(), JobType: "snapshot-csv"},
			}, &IngestionResult{
				Status:  "ok",
				RunID:   apiRunID.String(),
				JobType: "snapshot-api",
			})

			output, err := h.Handle(ctx, event)

			Expect(err).NotTo(HaveOccurred())
			Expect(output.Status).To(Equal("ok"))
			Expect(upsertedOpps).To(HaveLen(1))
			Expect(upsertedOpps[0].Title).To(Equal("CSV Version"))
			Expect(reconcileDQ).To(HaveLen(1))
			Expect(reconcileDQ[0].FieldName).To(Equal("Title"))
			Expect(reconcileDQ[0].CSVValue).To(Equal("CSV Version"))
			Expect(reconcileDQ[0].APIValue).To(Equal("API Version"))
		})

		It("tracks DQ issues from API data with source 'api'", func() {
			store.GetSnapAPIRawDataFn = func(_ context.Context, _ uuid.UUID) ([]database.SnapAPIRawRow, error) {
				return []database.SnapAPIRawRow{
					{
						NoticeID: "DQ-API-001",
						RawData:  json.RawMessage(`{"noticeId":"DQ-API-001","active":"Yes","archiveDate":"1969-12-31"}`),
					},
				}, nil
			}

			var insertedIssues []database.DataQualityEntry
			store.InsertDataQualityIssuesFn = func(_ context.Context, _ uuid.UUID, entries []database.DataQualityEntry) {
				insertedIssues = entries
			}

			store.UpsertOpportunitiesFn = func(_ context.Context, _ uuid.UUID, _ time.Time, opps []reconcile.Opportunity) (int, error) {
				return len(opps), nil
			}

			event := buildEvent([]IngestionResult{}, &IngestionResult{
				Status:  "ok",
				RunID:   apiRunID.String(),
				JobType: "snapshot-api",
			})

			output, err := h.Handle(ctx, event)

			Expect(err).NotTo(HaveOccurred())
			Expect(output.Status).To(Equal("ok"))
			Expect(insertedIssues).To(HaveLen(1))
			Expect(insertedIssues[0].Source).To(Equal("api"))
			Expect(insertedIssues[0].FieldName).To(Equal("ArchiveDate"))
			Expect(insertedIssues[0].IssueType).To(Equal("sentinel_date"))
		})

		It("handles API-only with no CSV ingestion results", func() {
			store.GetSnapAPIRawDataFn = func(_ context.Context, _ uuid.UUID) ([]database.SnapAPIRawRow, error) {
				return []database.SnapAPIRawRow{
					{
						NoticeID: "API-ONLY-001",
						RawData:  json.RawMessage(`{"noticeId":"API-ONLY-001","title":"API Only","active":"Yes"}`),
					},
					{
						NoticeID: "API-ONLY-002",
						RawData:  json.RawMessage(`{"noticeId":"API-ONLY-002","title":"API Only 2","active":"Yes"}`),
					},
				}, nil
			}

			var upsertedOpps []reconcile.Opportunity
			store.UpsertOpportunitiesFn = func(_ context.Context, _ uuid.UUID, _ time.Time, opps []reconcile.Opportunity) (int, error) {
				upsertedOpps = opps
				return len(opps), nil
			}

			event := buildEvent([]IngestionResult{}, &IngestionResult{
				Status:  "ok",
				RunID:   apiRunID.String(),
				JobType: "snapshot-api",
			})

			output, err := h.Handle(ctx, event)

			Expect(err).NotTo(HaveOccurred())
			Expect(output.Status).To(Equal("ok"))
			Expect(upsertedOpps).To(HaveLen(2))
		})
	})

	Context("change detection optimization", func() {
		var activeRunID uuid.UUID

		BeforeEach(func() {
			activeRunID = uuid.New()
		})

		It("sends only changed records to UpsertOpportunities and touches unchanged", func() {
			// Build a known CSV opp so we can compute its hash
			csvRow := map[string]string{"NoticeId": "UNCHANGED-001", "Title": "Same Title", "Active": "Yes"}
			unchangedOpp, _ := reconcile.FromCSV(csvRow)
			unchangedOpp.DataSources = "csv"
			unchangedHash := unchangedOpp.ContentHash()

			store.GetSnapCSVRawDataFn = func(_ context.Context, _ uuid.UUID) ([]map[string]string, error) {
				return []map[string]string{
					csvRow,
					{"NoticeId": "CHANGED-001", "Title": "New Title", "Active": "Yes"},
					{"NoticeId": "NEW-001", "Title": "Brand New", "Active": "Yes"},
				}, nil
			}

			store.GetExistingOpportunityHashesFn = func(_ context.Context) (map[string]string, error) {
				return map[string]string{
					"UNCHANGED-001": unchangedHash,
					"CHANGED-001":   "old-hash-that-no-longer-matches",
				}, nil
			}

			var upsertedOpps []reconcile.Opportunity
			store.UpsertOpportunitiesFn = func(_ context.Context, _ uuid.UUID, _ time.Time, opps []reconcile.Opportunity) (int, error) {
				upsertedOpps = opps
				return len(opps), nil
			}

			var touchedRecords []database.UnchangedRecord
			store.BulkTouchUnchangedFn = func(_ context.Context, _ uuid.UUID, _ time.Time, records []database.UnchangedRecord) (int, error) {
				touchedRecords = records
				return len(records), nil
			}

			event := buildEvent([]IngestionResult{
				{Status: "ok", RunID: activeRunID.String(), JobType: "snapshot-csv"},
			}, nil)

			output, err := h.Handle(ctx, event)

			Expect(err).NotTo(HaveOccurred())
			Expect(output.Status).To(Equal("ok"))

			// Only changed + new should go to upsert
			Expect(upsertedOpps).To(HaveLen(2))
			upsertIDs := make([]string, len(upsertedOpps))
			for i, o := range upsertedOpps {
				upsertIDs[i] = o.NoticeID
			}
			Expect(upsertIDs).To(ConsistOf("CHANGED-001", "NEW-001"))

			// Unchanged should be touched
			Expect(touchedRecords).To(HaveLen(1))
			Expect(touchedRecords[0].NoticeID).To(Equal("UNCHANGED-001"))
			Expect(touchedRecords[0].Active).To(BeTrue())
			Expect(touchedRecords[0].DataSources).To(Equal("csv"))
		})

		It("falls back to full upsert when GetExistingOpportunityHashes fails", func() {
			store.GetSnapCSVRawDataFn = func(_ context.Context, _ uuid.UUID) ([]map[string]string, error) {
				return []map[string]string{
					{"NoticeId": "OPP-001", "Title": "Test", "Active": "Yes"},
				}, nil
			}

			store.GetExistingOpportunityHashesFn = func(_ context.Context) (map[string]string, error) {
				return nil, errors.New("connection timeout")
			}

			var upsertedOpps []reconcile.Opportunity
			store.UpsertOpportunitiesFn = func(_ context.Context, _ uuid.UUID, _ time.Time, opps []reconcile.Opportunity) (int, error) {
				upsertedOpps = opps
				return len(opps), nil
			}

			touchCalled := false
			store.BulkTouchUnchangedFn = func(_ context.Context, _ uuid.UUID, _ time.Time, _ []database.UnchangedRecord) (int, error) {
				touchCalled = true
				return 0, nil
			}

			event := buildEvent([]IngestionResult{
				{Status: "ok", RunID: activeRunID.String(), JobType: "snapshot-csv"},
			}, nil)

			output, err := h.Handle(ctx, event)

			Expect(err).NotTo(HaveOccurred())
			Expect(output.Status).To(Equal("ok"))
			Expect(upsertedOpps).To(HaveLen(1))
			Expect(touchCalled).To(BeFalse())
		})

		It("treats all records as changed when no existing hashes match", func() {
			store.GetSnapCSVRawDataFn = func(_ context.Context, _ uuid.UUID) ([]map[string]string, error) {
				return []map[string]string{
					{"NoticeId": "NEW-001", "Title": "A", "Active": "Yes"},
					{"NoticeId": "NEW-002", "Title": "B", "Active": "Yes"},
				}, nil
			}

			store.GetExistingOpportunityHashesFn = func(_ context.Context) (map[string]string, error) {
				return map[string]string{}, nil
			}

			var upsertedOpps []reconcile.Opportunity
			store.UpsertOpportunitiesFn = func(_ context.Context, _ uuid.UUID, _ time.Time, opps []reconcile.Opportunity) (int, error) {
				upsertedOpps = opps
				return len(opps), nil
			}

			touchCalled := false
			store.BulkTouchUnchangedFn = func(_ context.Context, _ uuid.UUID, _ time.Time, _ []database.UnchangedRecord) (int, error) {
				touchCalled = true
				return 0, nil
			}

			event := buildEvent([]IngestionResult{
				{Status: "ok", RunID: activeRunID.String(), JobType: "snapshot-csv"},
			}, nil)

			output, err := h.Handle(ctx, event)

			Expect(err).NotTo(HaveOccurred())
			Expect(output.Status).To(Equal("ok"))
			Expect(upsertedOpps).To(HaveLen(2))
			Expect(touchCalled).To(BeFalse())
		})

		It("reports touched count in pipeline step stats", func() {
			csvRow := map[string]string{"NoticeId": "UNCHANGED-001", "Title": "Same", "Active": "Yes"}
			unchangedOpp, _ := reconcile.FromCSV(csvRow)
			unchangedOpp.DataSources = "csv"

			store.GetSnapCSVRawDataFn = func(_ context.Context, _ uuid.UUID) ([]map[string]string, error) {
				return []map[string]string{csvRow}, nil
			}

			store.GetExistingOpportunityHashesFn = func(_ context.Context) (map[string]string, error) {
				return map[string]string{
					"UNCHANGED-001": unchangedOpp.ContentHash(),
				}, nil
			}

			store.BulkTouchUnchangedFn = func(_ context.Context, _ uuid.UUID, _ time.Time, records []database.UnchangedRecord) (int, error) {
				return len(records), nil
			}

			var capturedStats map[string]any
			store.CompletePipelineStepFn = func(_ context.Context, _ uuid.UUID, stats map[string]any, _ int) error {
				capturedStats = stats
				return nil
			}

			execID := uuid.New()
			input := Input{
				ExecutionID: execID.String(),
				IngestionResults: []IngestionResult{
					{Status: "ok", RunID: activeRunID.String(), JobType: "snapshot-csv"},
				},
			}
			b, _ := json.Marshal(input)

			output, err := h.Handle(ctx, b)

			Expect(err).NotTo(HaveOccurred())
			Expect(output.Status).To(Equal("ok"))
			Expect(capturedStats).NotTo(BeNil())
			Expect(capturedStats["upserted"]).To(Equal(0))
			Expect(capturedStats["touched"]).To(Equal(1))
		})
	})
})
