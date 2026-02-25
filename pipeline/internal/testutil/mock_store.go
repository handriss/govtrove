package testutil

import (
	"context"
	"log/slog"
	"time"

	"github.com/google/uuid"
	"github.com/handriss/govtrove/pipeline/internal/database"
	"github.com/handriss/govtrove/pipeline/internal/reconcile"
)

type MockStore struct {
	CreatePipelineRunFn   func(ctx context.Context, name string, metadata map[string]any) (uuid.UUID, error)
	CompletePipelineRunFn func(ctx context.Context, id uuid.UUID, stats map[string]any, durationMs int) error
	FailPipelineRunFn     func(ctx context.Context, id uuid.UUID, errMsg string, durationMs int) error

	CreateIngestionRunFn   func(ctx context.Context, jobType string) (uuid.UUID, error)
	CompleteIngestionRunFn func(ctx context.Context, runID uuid.UUID, s database.RunStats) error
	FailIngestionRunFn     func(ctx context.Context, runID uuid.UUID, errMsg string, durationMs int) error
	GetLastCompletedRunFn  func(ctx context.Context, jobType string) (uuid.UUID, time.Time, error)

	CreateCSVDownloadEntryFn   func(ctx context.Context, e *database.CSVDownloadEntry) (int64, error)
	CompleteCSVDownloadEntryFn func(ctx context.Context, id int64, recordCount int, fileSizeBytes int64) error
	FailCSVDownloadEntryFn     func(ctx context.Context, id int64, errMsg string) error

	BulkInsertSnapCSVFn     func(ctx context.Context, runID uuid.UUID, snapshotDate time.Time, downloadID int64, rows []database.SnapCSVRow) (int64, error)
	DetectChangesFn         func(ctx context.Context, currentRunID, previousRunID uuid.UUID, snapshotDate time.Time, logger *slog.Logger) (int, int, error)
	DetectDisappearancesFn  func(ctx context.Context, currentRunID, previousRunID uuid.UUID, snapshotDate time.Time, logger *slog.Logger) (int, error)
	DetectReappearancesFn   func(ctx context.Context, currentRunID uuid.UUID, snapshotDate time.Time) (int, error)

	GetSnapCSVRawDataFn              func(ctx context.Context, runID uuid.UUID) ([]map[string]string, error)
	InsertDataQualityIssuesFn        func(ctx context.Context, runID uuid.UUID, entries []database.DataQualityEntry)
	UpsertOpportunitiesFn            func(ctx context.Context, runID uuid.UUID, snapshotDate time.Time, opps []reconcile.Opportunity) (int, error)
	ResolveExpectedDisappearancesFn  func(ctx context.Context, activeRunID, archivedRunID uuid.UUID, snapshotDate time.Time) (int, error)
	MarkDisappearedInactiveFn        func(ctx context.Context, runID uuid.UUID) (int, error)

	BulkInsertSnapAPIFn            func(ctx context.Context, runID uuid.UUID, snapshotDate time.Time, rows []database.SnapAPIRow) (int64, error)
	UpsertOpportunitiesFromAPIFn   func(ctx context.Context, runID uuid.UUID, snapshotDate time.Time, opps []reconcile.Opportunity) (int, error)

	InsertBulkCSVLogFn        func(ctx context.Context, r *database.BulkCSVLogRecord) (int, error)
	GetLatestBulkCSVHashFn    func(ctx context.Context, source string) (string, error)
	GetLatestBulkCSVHeadersFn func(ctx context.Context, source string) (string, string, error)
	GetLatestBulkCSVS3KeyFn   func(ctx context.Context, source string) (string, error)

	DeleteOldSearchEventsFn func(ctx context.Context, days int) (int64, error)
}

var _ database.Store = (*MockStore)(nil)

func (m *MockStore) CreatePipelineRun(ctx context.Context, name string, metadata map[string]any) (uuid.UUID, error) {
	if m.CreatePipelineRunFn != nil {
		return m.CreatePipelineRunFn(ctx, name, metadata)
	}
	return uuid.New(), nil
}

func (m *MockStore) CompletePipelineRun(ctx context.Context, id uuid.UUID, stats map[string]any, durationMs int) error {
	if m.CompletePipelineRunFn != nil {
		return m.CompletePipelineRunFn(ctx, id, stats, durationMs)
	}
	return nil
}

func (m *MockStore) FailPipelineRun(ctx context.Context, id uuid.UUID, errMsg string, durationMs int) error {
	if m.FailPipelineRunFn != nil {
		return m.FailPipelineRunFn(ctx, id, errMsg, durationMs)
	}
	return nil
}

func (m *MockStore) CreateIngestionRun(ctx context.Context, jobType string) (uuid.UUID, error) {
	if m.CreateIngestionRunFn != nil {
		return m.CreateIngestionRunFn(ctx, jobType)
	}
	return uuid.New(), nil
}

func (m *MockStore) CompleteIngestionRun(ctx context.Context, runID uuid.UUID, s database.RunStats) error {
	if m.CompleteIngestionRunFn != nil {
		return m.CompleteIngestionRunFn(ctx, runID, s)
	}
	return nil
}

func (m *MockStore) FailIngestionRun(ctx context.Context, runID uuid.UUID, errMsg string, durationMs int) error {
	if m.FailIngestionRunFn != nil {
		return m.FailIngestionRunFn(ctx, runID, errMsg, durationMs)
	}
	return nil
}

func (m *MockStore) GetLastCompletedRun(ctx context.Context, jobType string) (uuid.UUID, time.Time, error) {
	if m.GetLastCompletedRunFn != nil {
		return m.GetLastCompletedRunFn(ctx, jobType)
	}
	return uuid.Nil, time.Time{}, nil
}

func (m *MockStore) CreateCSVDownloadEntry(ctx context.Context, e *database.CSVDownloadEntry) (int64, error) {
	if m.CreateCSVDownloadEntryFn != nil {
		return m.CreateCSVDownloadEntryFn(ctx, e)
	}
	return 1, nil
}

func (m *MockStore) CompleteCSVDownloadEntry(ctx context.Context, id int64, recordCount int, fileSizeBytes int64) error {
	if m.CompleteCSVDownloadEntryFn != nil {
		return m.CompleteCSVDownloadEntryFn(ctx, id, recordCount, fileSizeBytes)
	}
	return nil
}

func (m *MockStore) FailCSVDownloadEntry(ctx context.Context, id int64, errMsg string) error {
	if m.FailCSVDownloadEntryFn != nil {
		return m.FailCSVDownloadEntryFn(ctx, id, errMsg)
	}
	return nil
}

func (m *MockStore) BulkInsertSnapCSV(ctx context.Context, runID uuid.UUID, snapshotDate time.Time, downloadID int64, rows []database.SnapCSVRow) (int64, error) {
	if m.BulkInsertSnapCSVFn != nil {
		return m.BulkInsertSnapCSVFn(ctx, runID, snapshotDate, downloadID, rows)
	}
	return int64(len(rows)), nil
}

func (m *MockStore) DetectChanges(ctx context.Context, currentRunID, previousRunID uuid.UUID, snapshotDate time.Time, logger *slog.Logger) (int, int, error) {
	if m.DetectChangesFn != nil {
		return m.DetectChangesFn(ctx, currentRunID, previousRunID, snapshotDate, logger)
	}
	return 0, 0, nil
}

func (m *MockStore) DetectDisappearances(ctx context.Context, currentRunID, previousRunID uuid.UUID, snapshotDate time.Time, logger *slog.Logger) (int, error) {
	if m.DetectDisappearancesFn != nil {
		return m.DetectDisappearancesFn(ctx, currentRunID, previousRunID, snapshotDate, logger)
	}
	return 0, nil
}

func (m *MockStore) DetectReappearances(ctx context.Context, currentRunID uuid.UUID, snapshotDate time.Time) (int, error) {
	if m.DetectReappearancesFn != nil {
		return m.DetectReappearancesFn(ctx, currentRunID, snapshotDate)
	}
	return 0, nil
}

func (m *MockStore) GetSnapCSVRawData(ctx context.Context, runID uuid.UUID) ([]map[string]string, error) {
	if m.GetSnapCSVRawDataFn != nil {
		return m.GetSnapCSVRawDataFn(ctx, runID)
	}
	return nil, nil
}

func (m *MockStore) InsertDataQualityIssues(ctx context.Context, runID uuid.UUID, entries []database.DataQualityEntry) {
	if m.InsertDataQualityIssuesFn != nil {
		m.InsertDataQualityIssuesFn(ctx, runID, entries)
	}
}

func (m *MockStore) UpsertOpportunities(ctx context.Context, runID uuid.UUID, snapshotDate time.Time, opps []reconcile.Opportunity) (int, error) {
	if m.UpsertOpportunitiesFn != nil {
		return m.UpsertOpportunitiesFn(ctx, runID, snapshotDate, opps)
	}
	return len(opps), nil
}

func (m *MockStore) ResolveExpectedDisappearances(ctx context.Context, activeRunID, archivedRunID uuid.UUID, snapshotDate time.Time) (int, error) {
	if m.ResolveExpectedDisappearancesFn != nil {
		return m.ResolveExpectedDisappearancesFn(ctx, activeRunID, archivedRunID, snapshotDate)
	}
	return 0, nil
}

func (m *MockStore) MarkDisappearedInactive(ctx context.Context, runID uuid.UUID) (int, error) {
	if m.MarkDisappearedInactiveFn != nil {
		return m.MarkDisappearedInactiveFn(ctx, runID)
	}
	return 0, nil
}

func (m *MockStore) BulkInsertSnapAPI(ctx context.Context, runID uuid.UUID, snapshotDate time.Time, rows []database.SnapAPIRow) (int64, error) {
	if m.BulkInsertSnapAPIFn != nil {
		return m.BulkInsertSnapAPIFn(ctx, runID, snapshotDate, rows)
	}
	return int64(len(rows)), nil
}

func (m *MockStore) UpsertOpportunitiesFromAPI(ctx context.Context, runID uuid.UUID, snapshotDate time.Time, opps []reconcile.Opportunity) (int, error) {
	if m.UpsertOpportunitiesFromAPIFn != nil {
		return m.UpsertOpportunitiesFromAPIFn(ctx, runID, snapshotDate, opps)
	}
	return len(opps), nil
}

func (m *MockStore) InsertBulkCSVLog(ctx context.Context, r *database.BulkCSVLogRecord) (int, error) {
	if m.InsertBulkCSVLogFn != nil {
		return m.InsertBulkCSVLogFn(ctx, r)
	}
	return 1, nil
}

func (m *MockStore) GetLatestBulkCSVHash(ctx context.Context, source string) (string, error) {
	if m.GetLatestBulkCSVHashFn != nil {
		return m.GetLatestBulkCSVHashFn(ctx, source)
	}
	return "", nil
}

func (m *MockStore) GetLatestBulkCSVHeaders(ctx context.Context, source string) (string, string, error) {
	if m.GetLatestBulkCSVHeadersFn != nil {
		return m.GetLatestBulkCSVHeadersFn(ctx, source)
	}
	return "", "", nil
}

func (m *MockStore) GetLatestBulkCSVS3Key(ctx context.Context, source string) (string, error) {
	if m.GetLatestBulkCSVS3KeyFn != nil {
		return m.GetLatestBulkCSVS3KeyFn(ctx, source)
	}
	return "", nil
}

func (m *MockStore) DeleteOldSearchEvents(ctx context.Context, days int) (int64, error) {
	if m.DeleteOldSearchEventsFn != nil {
		return m.DeleteOldSearchEventsFn(ctx, days)
	}
	return 0, nil
}
