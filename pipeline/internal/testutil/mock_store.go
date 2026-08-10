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
	CreatePipelineRunFn   func(ctx context.Context, id uuid.UUID, name string, metadata map[string]any) (uuid.UUID, error)
	GetPipelineRunFn      func(ctx context.Context, id uuid.UUID) (*database.PipelineRun, error)
	CompletePipelineRunFn func(ctx context.Context, id uuid.UUID, stats map[string]any, durationMs int) error
	FailPipelineRunFn     func(ctx context.Context, id uuid.UUID, errMsg string, durationMs int) error

	CreateIngestionRunFn   func(ctx context.Context, jobType string, pipelineRunID *uuid.UUID) (uuid.UUID, error)
	CompleteIngestionRunFn func(ctx context.Context, runID uuid.UUID, s database.RunStats) error
	FailIngestionRunFn     func(ctx context.Context, runID uuid.UUID, errMsg string, durationMs int) error
	GetLastCompletedRunFn  func(ctx context.Context, jobType string) (uuid.UUID, time.Time, error)

	BulkInsertSnapCSVFn      func(ctx context.Context, runID uuid.UUID, snapshotDate time.Time, downloadID int64, rows []database.SnapCSVRow) (int64, error)
	GetPreviousRunHashesFn   func(ctx context.Context, runID uuid.UUID) (map[string]database.PreviousRunRecord, error)
	InsertDisappearancesFn   func(ctx context.Context, runID uuid.UUID, snapshotDate time.Time, records []database.DisappearedRecord, logger *slog.Logger) (int, error)
	DetectReappearancesFn    func(ctx context.Context, currentRunID uuid.UUID, snapshotDate time.Time) (int, error)

	GetSnapCSVRawDataFn              func(ctx context.Context, runID uuid.UUID) ([]map[string]string, error)
	InsertDataQualityIssuesFn        func(ctx context.Context, runID uuid.UUID, entries []database.DataQualityEntry)
	InsertReconcileDQIssuesFn        func(ctx context.Context, csvRunID, apiRunID uuid.UUID, entries []database.ReconcileDQEntry)
	GetExistingOpportunityHashesFn   func(ctx context.Context) (map[string]string, error)
	BulkTouchUnchangedFn             func(ctx context.Context, runID uuid.UUID, snapshotDate time.Time, records []database.UnchangedRecord) (int, error)
	UpsertOpportunitiesFn            func(ctx context.Context, runID uuid.UUID, snapshotDate time.Time, opps []reconcile.Opportunity) (int, error)
	MarkDisappearedInactiveFn           func(ctx context.Context, runID uuid.UUID) (int, error)
	DeactivateExpiredOpportunitiesFn func(ctx context.Context) (int, int, error)
	AnalyzeOpportunitiesFn           func(ctx context.Context) error

	BulkInsertSnapAPIFn func(ctx context.Context, runID uuid.UUID, snapshotDate time.Time, rows []database.SnapAPIRow) (int64, error)
	GetSnapAPIRawDataFn func(ctx context.Context, runID uuid.UUID) ([]database.SnapAPIRawRow, error)

	InsertBulkCSVLogFn        func(ctx context.Context, r *database.BulkCSVLogRecord) (int, error)
	GetLatestBulkCSVHashFn    func(ctx context.Context, source string) (string, error)
	GetLatestBulkCSVHeadersFn func(ctx context.Context, source string) (string, string, error)
	GetLatestBulkCSVS3KeyFn   func(ctx context.Context, source string) (string, error)
	GetBulkCSVLogByS3KeyFn            func(ctx context.Context, s3Key string) (*database.BulkCSVLogRecord, error)
	UpdateBulkCSVLogIngestionFn       func(ctx context.Context, id int, ingestionRunID uuid.UUID, recordCount int, status string) error

	CreatePipelineStepFn   func(ctx context.Context, executionID uuid.UUID, stepName string) (uuid.UUID, error)
	CompletePipelineStepFn func(ctx context.Context, id uuid.UUID, stats map[string]any, durationMs int) error
	FailPipelineStepFn     func(ctx context.Context, id uuid.UUID, errMsg string, durationMs int) error

	RefreshAgenciesFn          func(ctx context.Context) (int, error)
	RefreshCodeCorrelationsFn  func(ctx context.Context) error

	DeleteOldSearchEventsFn func(ctx context.Context, days int) (int64, error)
	DeleteOldSnapshotsFn    func(ctx context.Context, snapDays, dqDays int) (int64, error)

	GetNAICSVolumeDeltaFn  func(ctx context.Context) ([]database.NAICSVolume, error)
	GetAgencyVolumeDeltaFn func(ctx context.Context) ([]database.AgencyVolume, error)
	GetRecentTitlesFn      func(ctx context.Context) ([]database.TitleRow, error)
	GetSetAsideCountsFn    func(ctx context.Context) ([]database.SetAsideCount, error)

	GetSEOPageCountsFn         func(ctx context.Context, recentSince time.Time) ([]database.SEOPageCount, error)
	GetSEOPageOpportunitiesFn  func(ctx context.Context, filterCol, filterVal string, limit int) ([]database.SEOOpportunity, error)
	GetSEOPageOpportunitiesByPrefixFn func(ctx context.Context, filterCol, prefix string, limit int) ([]database.SEOOpportunity, error)
	GetActiveAgenciesFn        func(ctx context.Context) ([]database.AgencyInfo, error)
}

var _ database.Store = (*MockStore)(nil)

func (m *MockStore) CreatePipelineRun(ctx context.Context, id uuid.UUID, name string, metadata map[string]any) (uuid.UUID, error) {
	if m.CreatePipelineRunFn != nil {
		return m.CreatePipelineRunFn(ctx, id, name, metadata)
	}
	if id != uuid.Nil {
		return id, nil
	}
	return uuid.New(), nil
}

func (m *MockStore) GetPipelineRun(ctx context.Context, id uuid.UUID) (*database.PipelineRun, error) {
	if m.GetPipelineRunFn != nil {
		return m.GetPipelineRunFn(ctx, id)
	}
	return nil, nil
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

func (m *MockStore) CreateIngestionRun(ctx context.Context, jobType string, pipelineRunID *uuid.UUID) (uuid.UUID, error) {
	if m.CreateIngestionRunFn != nil {
		return m.CreateIngestionRunFn(ctx, jobType, pipelineRunID)
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

func (m *MockStore) BulkInsertSnapCSV(ctx context.Context, runID uuid.UUID, snapshotDate time.Time, downloadID int64, rows []database.SnapCSVRow) (int64, error) {
	if m.BulkInsertSnapCSVFn != nil {
		return m.BulkInsertSnapCSVFn(ctx, runID, snapshotDate, downloadID, rows)
	}
	return int64(len(rows)), nil
}

func (m *MockStore) GetPreviousRunHashes(ctx context.Context, runID uuid.UUID) (map[string]database.PreviousRunRecord, error) {
	if m.GetPreviousRunHashesFn != nil {
		return m.GetPreviousRunHashesFn(ctx, runID)
	}
	return nil, nil
}

func (m *MockStore) InsertDisappearances(ctx context.Context, runID uuid.UUID, snapshotDate time.Time, records []database.DisappearedRecord, logger *slog.Logger) (int, error) {
	if m.InsertDisappearancesFn != nil {
		return m.InsertDisappearancesFn(ctx, runID, snapshotDate, records, logger)
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

func (m *MockStore) InsertReconcileDQIssues(ctx context.Context, csvRunID, apiRunID uuid.UUID, entries []database.ReconcileDQEntry) {
	if m.InsertReconcileDQIssuesFn != nil {
		m.InsertReconcileDQIssuesFn(ctx, csvRunID, apiRunID, entries)
	}
}

func (m *MockStore) GetExistingOpportunityHashes(ctx context.Context) (map[string]string, error) {
	if m.GetExistingOpportunityHashesFn != nil {
		return m.GetExistingOpportunityHashesFn(ctx)
	}
	return nil, nil
}

func (m *MockStore) BulkTouchUnchanged(ctx context.Context, runID uuid.UUID, snapshotDate time.Time, records []database.UnchangedRecord) (int, error) {
	if m.BulkTouchUnchangedFn != nil {
		return m.BulkTouchUnchangedFn(ctx, runID, snapshotDate, records)
	}
	return len(records), nil
}

func (m *MockStore) UpsertOpportunities(ctx context.Context, runID uuid.UUID, snapshotDate time.Time, opps []reconcile.Opportunity) (int, error) {
	if m.UpsertOpportunitiesFn != nil {
		return m.UpsertOpportunitiesFn(ctx, runID, snapshotDate, opps)
	}
	return len(opps), nil
}

func (m *MockStore) MarkDisappearedInactive(ctx context.Context, runID uuid.UUID) (int, error) {
	if m.MarkDisappearedInactiveFn != nil {
		return m.MarkDisappearedInactiveFn(ctx, runID)
	}
	return 0, nil
}

func (m *MockStore) DeactivateExpiredOpportunities(ctx context.Context) (int, int, error) {
	if m.DeactivateExpiredOpportunitiesFn != nil {
		return m.DeactivateExpiredOpportunitiesFn(ctx)
	}
	return 0, 0, nil
}

func (m *MockStore) AnalyzeOpportunities(ctx context.Context) error {
	if m.AnalyzeOpportunitiesFn != nil {
		return m.AnalyzeOpportunitiesFn(ctx)
	}
	return nil
}

func (m *MockStore) BulkInsertSnapAPI(ctx context.Context, runID uuid.UUID, snapshotDate time.Time, rows []database.SnapAPIRow) (int64, error) {
	if m.BulkInsertSnapAPIFn != nil {
		return m.BulkInsertSnapAPIFn(ctx, runID, snapshotDate, rows)
	}
	return int64(len(rows)), nil
}

func (m *MockStore) GetSnapAPIRawData(ctx context.Context, runID uuid.UUID) ([]database.SnapAPIRawRow, error) {
	if m.GetSnapAPIRawDataFn != nil {
		return m.GetSnapAPIRawDataFn(ctx, runID)
	}
	return nil, nil
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

func (m *MockStore) GetBulkCSVLogByS3Key(ctx context.Context, s3Key string) (*database.BulkCSVLogRecord, error) {
	if m.GetBulkCSVLogByS3KeyFn != nil {
		return m.GetBulkCSVLogByS3KeyFn(ctx, s3Key)
	}
	return nil, nil
}

func (m *MockStore) UpdateBulkCSVLogIngestion(ctx context.Context, id int, ingestionRunID uuid.UUID, recordCount int, status string) error {
	if m.UpdateBulkCSVLogIngestionFn != nil {
		return m.UpdateBulkCSVLogIngestionFn(ctx, id, ingestionRunID, recordCount, status)
	}
	return nil
}

func (m *MockStore) RefreshAgencies(ctx context.Context) (int, error) {
	if m.RefreshAgenciesFn != nil {
		return m.RefreshAgenciesFn(ctx)
	}
	return 0, nil
}

func (m *MockStore) RefreshCodeCorrelations(ctx context.Context) error {
	if m.RefreshCodeCorrelationsFn != nil {
		return m.RefreshCodeCorrelationsFn(ctx)
	}
	return nil
}

func (m *MockStore) CreatePipelineStep(ctx context.Context, executionID uuid.UUID, stepName string) (uuid.UUID, error) {
	if m.CreatePipelineStepFn != nil {
		return m.CreatePipelineStepFn(ctx, executionID, stepName)
	}
	return uuid.New(), nil
}

func (m *MockStore) CompletePipelineStep(ctx context.Context, id uuid.UUID, stats map[string]any, durationMs int) error {
	if m.CompletePipelineStepFn != nil {
		return m.CompletePipelineStepFn(ctx, id, stats, durationMs)
	}
	return nil
}

func (m *MockStore) FailPipelineStep(ctx context.Context, id uuid.UUID, errMsg string, durationMs int) error {
	if m.FailPipelineStepFn != nil {
		return m.FailPipelineStepFn(ctx, id, errMsg, durationMs)
	}
	return nil
}

func (m *MockStore) DeleteOldSearchEvents(ctx context.Context, days int) (int64, error) {
	if m.DeleteOldSearchEventsFn != nil {
		return m.DeleteOldSearchEventsFn(ctx, days)
	}
	return 0, nil
}

func (m *MockStore) DeleteOldSnapshots(ctx context.Context, snapDays, dqDays int) (int64, error) {
	if m.DeleteOldSnapshotsFn != nil {
		return m.DeleteOldSnapshotsFn(ctx, snapDays, dqDays)
	}
	return 0, nil
}

func (m *MockStore) GetNAICSVolumeDelta(ctx context.Context) ([]database.NAICSVolume, error) {
	if m.GetNAICSVolumeDeltaFn != nil {
		return m.GetNAICSVolumeDeltaFn(ctx)
	}
	return nil, nil
}

func (m *MockStore) GetAgencyVolumeDelta(ctx context.Context) ([]database.AgencyVolume, error) {
	if m.GetAgencyVolumeDeltaFn != nil {
		return m.GetAgencyVolumeDeltaFn(ctx)
	}
	return nil, nil
}

func (m *MockStore) GetRecentTitles(ctx context.Context) ([]database.TitleRow, error) {
	if m.GetRecentTitlesFn != nil {
		return m.GetRecentTitlesFn(ctx)
	}
	return nil, nil
}

func (m *MockStore) GetSetAsideCounts(ctx context.Context) ([]database.SetAsideCount, error) {
	if m.GetSetAsideCountsFn != nil {
		return m.GetSetAsideCountsFn(ctx)
	}
	return nil, nil
}

func (m *MockStore) GetSEOPageCounts(ctx context.Context, recentSince time.Time) ([]database.SEOPageCount, error) {
	if m.GetSEOPageCountsFn != nil {
		return m.GetSEOPageCountsFn(ctx, recentSince)
	}
	return nil, nil
}

func (m *MockStore) GetSEOPageOpportunities(ctx context.Context, filterCol, filterVal string, limit int) ([]database.SEOOpportunity, error) {
	if m.GetSEOPageOpportunitiesFn != nil {
		return m.GetSEOPageOpportunitiesFn(ctx, filterCol, filterVal, limit)
	}
	return nil, nil
}

func (m *MockStore) GetSEOPageOpportunitiesByPrefix(ctx context.Context, filterCol, prefix string, limit int) ([]database.SEOOpportunity, error) {
	if m.GetSEOPageOpportunitiesByPrefixFn != nil {
		return m.GetSEOPageOpportunitiesByPrefixFn(ctx, filterCol, prefix, limit)
	}
	return nil, nil
}

func (m *MockStore) GetActiveAgencies(ctx context.Context) ([]database.AgencyInfo, error) {
	if m.GetActiveAgenciesFn != nil {
		return m.GetActiveAgenciesFn(ctx)
	}
	return nil, nil
}
