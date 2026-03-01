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
			PipelineRunID:    uuid.New().String(),
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
})
