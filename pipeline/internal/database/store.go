package database

import (
	"context"
	"log/slog"
	"time"

	"github.com/google/uuid"
	"github.com/handriss/govtrove/pipeline/internal/reconcile"
)

type Store interface {
	// Pipeline runs
	CreatePipelineRun(ctx context.Context, id uuid.UUID, name string, metadata map[string]any) (uuid.UUID, error)
	GetPipelineRun(ctx context.Context, id uuid.UUID) (*PipelineRun, error)
	CompletePipelineRun(ctx context.Context, id uuid.UUID, stats map[string]any, durationMs int) error
	FailPipelineRun(ctx context.Context, id uuid.UUID, errMsg string, durationMs int) error

	// Ingestion runs
	CreateIngestionRun(ctx context.Context, jobType string, pipelineRunID *uuid.UUID) (uuid.UUID, error)
	CompleteIngestionRun(ctx context.Context, runID uuid.UUID, s RunStats) error
	FailIngestionRun(ctx context.Context, runID uuid.UUID, errMsg string, durationMs int) error
	GetLastCompletedRun(ctx context.Context, jobType string) (uuid.UUID, time.Time, error)

	// Snapshot operations
	BulkInsertSnapCSV(ctx context.Context, runID uuid.UUID, snapshotDate time.Time, downloadID int64, rows []SnapCSVRow) (int64, error)
	GetPreviousRunHashes(ctx context.Context, runID uuid.UUID) (map[string]PreviousRunRecord, error)
	InsertDisappearances(ctx context.Context, runID uuid.UUID, snapshotDate time.Time, records []DisappearedRecord, logger *slog.Logger) (int, error)
	DetectReappearances(ctx context.Context, currentRunID uuid.UUID, snapshotDate time.Time) (int, error)

	// Reconcile operations
	GetSnapCSVRawData(ctx context.Context, runID uuid.UUID) ([]map[string]string, error)
	InsertDataQualityIssues(ctx context.Context, runID uuid.UUID, entries []DataQualityEntry)
	InsertReconcileDQIssues(ctx context.Context, csvRunID, apiRunID uuid.UUID, entries []ReconcileDQEntry)
	GetExistingOpportunityHashes(ctx context.Context) (map[string]string, error)
	BulkTouchUnchanged(ctx context.Context, runID uuid.UUID, snapshotDate time.Time, records []UnchangedRecord) (int, error)
	UpsertOpportunities(ctx context.Context, runID uuid.UUID, snapshotDate time.Time, opps []reconcile.Opportunity) (int, error)
	MarkDisappearedInactive(ctx context.Context, runID uuid.UUID) (int, error)
	DeactivateExpiredOpportunities(ctx context.Context) (int, int, error)

	// API snapshot operations
	BulkInsertSnapAPI(ctx context.Context, runID uuid.UUID, snapshotDate time.Time, rows []SnapAPIRow) (int64, error)
	GetSnapAPIRawData(ctx context.Context, runID uuid.UUID) ([]SnapAPIRawRow, error)

	// Bulk CSV log
	InsertBulkCSVLog(ctx context.Context, r *BulkCSVLogRecord) (int, error)
	GetLatestBulkCSVHash(ctx context.Context, source string) (string, error)
	GetLatestBulkCSVHeaders(ctx context.Context, source string) (string, string, error)
	GetLatestBulkCSVS3Key(ctx context.Context, source string) (string, error)
	GetBulkCSVLogByS3Key(ctx context.Context, s3Key string) (*BulkCSVLogRecord, error)
	UpdateBulkCSVLogIngestion(ctx context.Context, id int, ingestionRunID uuid.UUID, recordCount int, status string) error

	// Agencies
	RefreshAgencies(ctx context.Context) (int, error)

	// Pipeline steps
	CreatePipelineStep(ctx context.Context, executionID uuid.UUID, stepName string) (uuid.UUID, error)
	CompletePipelineStep(ctx context.Context, id uuid.UUID, stats map[string]any, durationMs int) error
	FailPipelineStep(ctx context.Context, id uuid.UUID, errMsg string, durationMs int) error

	// Analytics retention
	DeleteOldSearchEvents(ctx context.Context, days int) (int64, error)
}

// Verify *DB satisfies Store at compile time.
var _ Store = (*DB)(nil)
