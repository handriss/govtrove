package ingestion

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/sns"
	"github.com/google/uuid"
	"github.com/handriss/govtrove/jobs/internal/config"
	"github.com/handriss/govtrove/jobs/internal/database"
	"github.com/handriss/govtrove/jobs/internal/samgov"
)

type Service struct {
	db     *database.DB
	csv    *samgov.CSVClient
	sns    *sns.Client
	cfg    *config.Config
	logger *slog.Logger
}

func New(cfg *config.Config, db *database.DB, snsClient *sns.Client, logger *slog.Logger) *Service {
	return &Service{
		cfg:    cfg,
		db:     db,
		sns:    snsClient,
		csv:    samgov.NewCSVClient(db, logger),
		logger: logger,
	}
}

func (s *Service) Run(ctx context.Context) error {
	startTime := time.Now()
	s.logger.Info("starting snapshot ingestion")

	runID, err := s.db.CreateIngestionRun(ctx, "snapshot-csv")
	if err != nil {
		return fmt.Errorf("create ingestion run: %w", err)
	}
	s.logger.Info("created ingestion run", "run_id", runID)

	snapshotDate := time.Now().UTC()
	stats := &snapshotStats{}

	err = s.runPipeline(ctx, runID, snapshotDate, stats)
	durationMs := int(time.Since(startTime).Milliseconds())

	if err != nil {
		s.db.FailIngestionRun(ctx, runID, err.Error(), durationMs)
		s.logger.Error("ingestion failed", "error", err, "duration_ms", durationMs)
		s.sendNotification(ctx, fmt.Sprintf("Ingest FAILED: %s", err))
		return err
	}

	runStats := database.RunStats{
		Fetched:    stats.recordCount,
		Inserted:   stats.newRecords,
		Updated:    stats.changedRecords,
		Failed:     0,
		Skipped:    0,
		DurationMs: durationMs,
	}
	if dbErr := s.db.CompleteIngestionRun(ctx, runID, runStats); dbErr != nil {
		s.logger.Error("failed to complete ingestion run", "error", dbErr)
		return dbErr
	}

	duration := time.Duration(durationMs) * time.Millisecond
	s.logger.Info("snapshot ingestion complete",
		"run_id", runID,
		"records", stats.recordCount,
		"new", stats.newRecords,
		"changed", stats.changedRecords,
		"disappeared", stats.disappearedRecords,
		"reappeared", stats.reappearedRecords,
		"duration", duration.Round(time.Second),
	)

	s.sendNotification(ctx, fmt.Sprintf(
		"Ingest — snapshot complete\nRecords: %d\nNew: %d\nChanged: %d\nDisappeared: %d\nDuration: %s",
		stats.recordCount, stats.newRecords, stats.changedRecords, stats.disappearedRecords,
		duration.Round(time.Second)))

	return nil
}

type snapshotStats struct {
	recordCount        int
	newRecords         int
	changedRecords     int
	disappearedRecords int
	reappearedRecords  int
}

func (s *Service) runPipeline(ctx context.Context, runID uuid.UUID, snapshotDate time.Time, stats *snapshotStats) error {
	// 1. Load cached ETag
	var etag, lastModified string
	cached, err := s.db.GetCSVCacheHeaders(ctx, samgov.FullCSVURL)
	if err == nil && cached != nil {
		etag = cached.ETag
		lastModified = cached.LastModified
	}

	// 2. Download CSV
	result, err := s.csv.FetchOpportunitiesConditional(ctx, s.cfg.RecordLimit, etag, lastModified)
	if err != nil {
		return fmt.Errorf("fetch CSV: %w", err)
	}

	// 3. Handle 304 Not Modified
	if result.NotModified {
		s.logger.Info("CSV not modified (304), skipping snapshot")
		if err := s.db.UpdateCSVCacheLastChecked(ctx, samgov.FullCSVURL); err != nil {
			s.logger.Warn("failed to update cache last_checked_at", "error", err)
		}
		return nil
	}

	// 4. Create download log entry
	downloadEntry := &database.CSVDownloadEntry{
		RunID:        runID,
		URL:          samgov.FullCSVURL,
		Status:       "downloading",
		ETag:         result.ETag,
		LastModified: result.LastModified,
	}
	downloadID, err := s.db.CreateCSVDownloadEntry(ctx, downloadEntry)
	if err != nil {
		return fmt.Errorf("create download entry: %w", err)
	}

	// 5. Convert rows to snapshot format
	s.logger.Info("converting CSV rows to snapshot format", "rows", result.ParsedRows)
	snapRows := make([]database.SnapCSVRow, 0, len(result.Rows))
	for _, raw := range result.Rows {
		snapRow := extractTypedFields(raw)
		snapRow.ContentHash = database.ComputeContentHash(raw)
		snapRows = append(snapRows, snapRow)
	}
	stats.recordCount = len(snapRows)

	// 6. Bulk insert via COPY protocol
	s.logger.Info("bulk inserting snapshot rows", "count", len(snapRows))
	inserted, err := s.db.BulkInsertSnapCSV(ctx, runID, snapshotDate, downloadID, snapRows)
	if err != nil {
		s.db.FailCSVDownloadEntry(ctx, downloadID, err.Error())
		return fmt.Errorf("bulk insert: %w", err)
	}
	s.logger.Info("bulk insert complete", "inserted", inserted)

	// 7. Complete download log
	s.db.CompleteCSVDownloadEntry(ctx, downloadID, len(snapRows), 0)

	// 8. Update cache headers
	if result.ETag != "" || result.LastModified != "" {
		now := time.Now()
		cacheHeaders := &database.CSVCacheHeaders{
			URL:              samgov.FullCSVURL,
			ETag:             result.ETag,
			LastModified:     result.LastModified,
			LastDownloadedAt: &now,
		}
		if err := s.db.UpsertCSVCacheHeaders(ctx, cacheHeaders); err != nil {
			s.logger.Warn("failed to save CSV cache headers", "error", err)
		}
	}

	// 9. Get previous run for change detection
	prevRunID, _, prevErr := s.db.GetLastCompletedRun(ctx, "snapshot-csv")
	if prevErr != nil {
		s.logger.Warn("failed to get previous run", "error", prevErr)
	}

	if prevRunID == uuid.Nil {
		s.logger.Info("first run, all records treated as new", "count", len(snapRows))
		stats.newRecords = len(snapRows)
		s.logger.Info("layer 3 reconciliation: not yet implemented, skipping")
		return nil
	}

	// 10. Change detection
	s.logger.Info("detecting changes", "current_run", runID, "previous_run", prevRunID)
	newCount, changedCount, err := s.db.DetectChanges(ctx, runID, prevRunID, snapshotDate, s.logger)
	if err != nil {
		s.logger.Error("change detection failed", "error", err)
	} else {
		stats.newRecords = newCount
		stats.changedRecords = changedCount
		s.logger.Info("change detection complete", "new", newCount, "changed", changedCount)
	}

	// 11. Disappearance detection
	disappearedCount, err := s.db.DetectDisappearances(ctx, runID, prevRunID, snapshotDate, s.logger)
	if err != nil {
		s.logger.Error("disappearance detection failed", "error", err)
	} else {
		stats.disappearedRecords = disappearedCount
		s.logger.Info("disappearance detection complete", "disappeared", disappearedCount)
	}

	// 12. Reappearance detection
	reappearedCount, err := s.db.DetectReappearances(ctx, runID, snapshotDate)
	if err != nil {
		s.logger.Error("reappearance detection failed", "error", err)
	} else {
		stats.reappearedRecords = reappearedCount
		if reappearedCount > 0 {
			s.logger.Info("reappearance detection complete", "reappeared", reappearedCount)
		}
	}

	// 13. Reconciliation stub
	s.logger.Info("layer 3 reconciliation: not yet implemented, skipping")

	return nil
}

func extractTypedFields(raw map[string]string) database.SnapCSVRow {
	return database.SnapCSVRow{
		NoticeID:           raw["NoticeId"],
		SolicitationNumber: raw["Sol#"],
		Title:              raw["Title"],
		Type:               raw["Type"],
		BaseType:           raw["BaseType"],
		PostedDate:         samgov.ParseDate(raw["PostedDate"]),
		ResponseDeadline:   samgov.ParseDate(raw["ResponseDeadLine"]),
		ArchiveDate:        raw["ArchiveDate"],
		ArchiveType:        raw["ArchiveType"],
		SetAsideCode:       raw["SetASideCode"],
		NAICSCode:          raw["NaicsCode"],
		ClassificationCode: raw["ClassificationCode"],
		Active:             samgov.ParseActive(raw["Active"]),

		Department: raw["Department/Ind.Agency"],
		SubTier:    raw["Sub-Tier"],
		Office:     raw["Office"],
		CGAC:       raw["CGAC"],
		FPDSCode:   raw["FPDS Code"],
		AACCode:    raw["AAC Code"],

		AwardNumber: raw["AwardNumber"],
		AwardDate:   raw["AwardDate"],
		AwardAmount: samgov.ParseAmount(raw["Award$"]),

		RawData: raw,
	}
}

func (s *Service) sendNotification(ctx context.Context, message string) {
	if s.sns == nil || s.cfg.SNSTopicARN == "" {
		return
	}
	_, err := s.sns.Publish(ctx, &sns.PublishInput{
		TopicArn: aws.String(s.cfg.SNSTopicARN),
		Subject:  aws.String("GovTrove Ingestion"),
		Message:  aws.String(message),
	})
	if err != nil {
		s.logger.Warn("failed to send notification", "error", err)
	}
}
