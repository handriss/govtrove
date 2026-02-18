package e2e_test

import (
	"compress/gzip"
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/handriss/govtrove/pipeline/internal/database"
	"github.com/handriss/govtrove/pipeline/internal/ingest"
	"github.com/handriss/govtrove/pipeline/internal/reconcile"
	"github.com/handriss/govtrove/pipeline/internal/samgov"
)

// simulateIngest mimics what the ingest-active and ingest-archived handlers do:
// create run, parse CSV, extract snap rows, bulk insert, and (for active) detect changes.
func simulateIngest(ctx context.Context, store database.Store, csvData string, jobType string, snapshotDate time.Time) (uuid.UUID, error) {
	runID, err := store.CreateIngestionRun(ctx, jobType)
	if err != nil {
		return uuid.Nil, fmt.Errorf("create ingestion run: %w", err)
	}

	gz := gzipCSV(csvData)
	gzReader, err := gzip.NewReader(gz)
	if err != nil {
		return uuid.Nil, fmt.Errorf("gzip reader: %w", err)
	}
	defer gzReader.Close()

	result, err := samgov.ParseCSVFromReader(gzReader, 0, logger)
	if err != nil {
		return uuid.Nil, fmt.Errorf("parse CSV: %w", err)
	}

	downloadID, err := store.CreateCSVDownloadEntry(ctx, &database.CSVDownloadEntry{
		RunID:  runID,
		URL:    "test://" + jobType,
		Status: "downloading",
	})
	if err != nil {
		return uuid.Nil, fmt.Errorf("create download entry: %w", err)
	}

	snapRows := make([]database.SnapCSVRow, 0, len(result.Rows))
	for _, raw := range result.Rows {
		snapRows = append(snapRows, ingest.ExtractSnapCSVRow(raw))
	}

	if _, err := store.BulkInsertSnapCSV(ctx, runID, snapshotDate, downloadID, snapRows); err != nil {
		return uuid.Nil, fmt.Errorf("bulk insert: %w", err)
	}

	_ = store.CompleteCSVDownloadEntry(ctx, downloadID, len(snapRows), 0)

	if jobType == "snapshot-csv" {
		prevRunID, _, prevErr := store.GetLastCompletedRun(ctx, jobType)
		if prevErr != nil {
			return uuid.Nil, fmt.Errorf("get previous run: %w", prevErr)
		}

		if prevRunID != uuid.Nil {
			if _, _, err := store.DetectChanges(ctx, runID, prevRunID, snapshotDate, logger); err != nil {
				return uuid.Nil, fmt.Errorf("detect changes: %w", err)
			}
			if _, err := store.DetectDisappearances(ctx, runID, prevRunID, snapshotDate, logger); err != nil {
				return uuid.Nil, fmt.Errorf("detect disappearances: %w", err)
			}
			if _, err := store.DetectReappearances(ctx, runID, snapshotDate); err != nil {
				return uuid.Nil, fmt.Errorf("detect reappearances: %w", err)
			}
		}
	}

	err = store.CompleteIngestionRun(ctx, runID, database.RunStats{
		Fetched:    len(snapRows),
		Inserted:   len(snapRows),
		DurationMs: 1,
	})
	if err != nil {
		return uuid.Nil, fmt.Errorf("complete ingestion run: %w", err)
	}

	return runID, nil
}

// simulateReconcile mimics what the reconcile handler does:
// load raw data, convert to opportunities, insert DQ issues, upsert, resolve disappearances, mark inactive.
func simulateReconcile(ctx context.Context, store database.Store, activeRunID, archivedRunID uuid.UUID, snapshotDate time.Time) error {
	for _, runID := range []uuid.UUID{activeRunID, archivedRunID} {
		if runID == uuid.Nil {
			continue
		}

		rows, err := store.GetSnapCSVRawData(ctx, runID)
		if err != nil {
			return fmt.Errorf("get raw data for %s: %w", runID, err)
		}

		opps := make([]reconcile.Opportunity, 0, len(rows))
		var dqEntries []database.DataQualityEntry
		for _, raw := range rows {
			opp, issues := reconcile.FromCSV(raw)
			opps = append(opps, opp)
			for _, iss := range issues {
				dqEntries = append(dqEntries, database.DataQualityEntry{
					NoticeID:     opp.NoticeID,
					SnapshotDate: snapshotDate,
					Source:       "csv",
					IssueType:    iss.IssueType,
					FieldName:    iss.FieldName,
					FieldValue:   iss.FieldValue,
				})
			}
		}

		if len(dqEntries) > 0 {
			store.InsertDataQualityIssues(ctx, runID, dqEntries)
		}

		if _, err := store.UpsertOpportunities(ctx, runID, snapshotDate, opps); err != nil {
			return fmt.Errorf("upsert from run %s: %w", runID, err)
		}
	}

	if activeRunID != uuid.Nil && archivedRunID != uuid.Nil {
		if _, err := store.ResolveExpectedDisappearances(ctx, activeRunID, archivedRunID, snapshotDate); err != nil {
			return fmt.Errorf("resolve disappearances: %w", err)
		}
	}

	if activeRunID != uuid.Nil {
		if _, err := store.MarkDisappearedInactive(ctx, activeRunID); err != nil {
			return fmt.Errorf("mark disappeared inactive: %w", err)
		}
	}

	return nil
}

// queryCount returns the count from a SQL query that returns a single integer.
func queryCount(ctx context.Context, sql string, args ...any) int {
	var count int
	err := db.Pool().QueryRow(ctx, sql, args...).Scan(&count)
	if err != nil {
		return -1
	}
	return count
}
