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

	buildEvent := func(results []IngestionResult) json.RawMessage {
		input := Input{
			PipelineRunID:    uuid.New().String(),
			IngestionResults: results,
		}
		b, _ := json.Marshal(input)
		return b
	}

	Context("with active and archived ingestion results", func() {
		var activeRunID, archivedRunID uuid.UUID

		BeforeEach(func() {
			activeRunID = uuid.New()
			archivedRunID = uuid.New()

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

		It("upserts opportunities from both runs and resolves disappearances", func() {
			event := buildEvent([]IngestionResult{
				{Status: "ok", RunID: activeRunID.String(), JobType: "snapshot-csv"},
				{Status: "ok", RunID: archivedRunID.String(), JobType: "ingest-archived"},
			})

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
			})

			output, err := h.Handle(ctx, event)

			Expect(err).NotTo(HaveOccurred())
			Expect(output.Status).To(Equal("ok"))
			Expect(insertedIssues).To(HaveLen(1))
			Expect(insertedIssues[0].IssueType).To(Equal("sentinel_date"))
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
			})

			_, err := h.Handle(ctx, event)

			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("failed"))
			Expect(upsertCalled).To(BeFalse())
		})
	})

	Context("when only active run is present (no archived)", func() {
		It("skips archived upsert and disappearance resolution", func() {
			activeRunID := uuid.New()

			store.GetSnapCSVRawDataFn = func(_ context.Context, _ uuid.UUID) ([]map[string]string, error) {
				return []map[string]string{
					{"NoticeId": "OPP-001", "Title": "Active Only"},
				}, nil
			}

			resolveCalled := false
			store.ResolveExpectedDisappearancesFn = func(_ context.Context, _, _ uuid.UUID, _ time.Time) (int, error) {
				resolveCalled = true
				return 0, nil
			}

			markCalled := false
			store.MarkDisappearedInactiveFn = func(_ context.Context, _ uuid.UUID) (int, error) {
				markCalled = true
				return 0, nil
			}

			event := buildEvent([]IngestionResult{
				{Status: "ok", RunID: activeRunID.String(), JobType: "snapshot-csv"},
			})

			output, err := h.Handle(ctx, event)

			Expect(err).NotTo(HaveOccurred())
			Expect(output.Status).To(Equal("ok"))
			Expect(resolveCalled).To(BeFalse())
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
			})

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
			})

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
			event := buildEvent([]IngestionResult{})

			output, err := h.Handle(ctx, event)

			Expect(err).NotTo(HaveOccurred())
			Expect(output.Status).To(Equal("ok"))
		})
	})
})
