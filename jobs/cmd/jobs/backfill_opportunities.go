package main

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/handriss/govtrove/jobs/internal/database"
	"github.com/handriss/govtrove/jobs/internal/reconcile"
)

func RunBackfillOpportunities(ctx context.Context, db *database.DB, logger *slog.Logger) error {
	startTime := time.Now()

	runID, startedAt, err := db.GetLastCompletedRun(ctx, "snapshot-csv")
	if err != nil {
		return fmt.Errorf("get last completed run: %w", err)
	}
	if runID.String() == "00000000-0000-0000-0000-000000000000" {
		return fmt.Errorf("no completed snapshot-csv run found")
	}
	logger.Info("loading snap_csv data", "run_id", runID, "started_at", startedAt)

	rows, err := db.GetSnapCSVRawData(ctx, runID)
	if err != nil {
		return fmt.Errorf("load snap_csv raw data: %w", err)
	}
	logger.Info("loaded rows from snap_csv", "count", len(rows))

	opps := make([]reconcile.Opportunity, 0, len(rows))
	for _, raw := range rows {
		opps = append(opps, reconcile.FromCSV(raw))
	}

	logger.Info("upserting opportunities", "count", len(opps))
	inserted, updated, err := db.UpsertOpportunities(ctx, runID, startedAt, opps)
	if err != nil {
		return fmt.Errorf("upsert opportunities: %w", err)
	}

	duration := time.Since(startTime)
	logger.Info("backfill complete",
		"upserted", inserted+updated,
		"duration", duration.Round(time.Second),
	)

	return nil
}
