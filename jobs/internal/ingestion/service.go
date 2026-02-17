package ingestion

import (
	"compress/gzip"
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/sns"
	"github.com/google/uuid"
	"github.com/handriss/govtrove/jobs/internal/config"
	"github.com/handriss/govtrove/jobs/internal/database"
	"github.com/handriss/govtrove/jobs/internal/reconcile"
	"github.com/handriss/govtrove/jobs/internal/samgov"
)

type Service struct {
	db       *database.DB
	csv      *samgov.CSVClient
	s3Client *s3.Client
	sns      *sns.Client
	cfg      *config.Config
	logger   *slog.Logger
}

func New(cfg *config.Config, db *database.DB, s3Client *s3.Client, snsClient *sns.Client, logger *slog.Logger) *Service {
	return &Service{
		cfg:      cfg,
		db:       db,
		s3Client: s3Client,
		sns:      snsClient,
		csv:      samgov.NewCSVClient(db, logger),
		logger:   logger,
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
		"Ingest — snapshot complete\nRecords: %d\nNew: %d\nChanged: %d\nDisappeared: %d\nOpps upserted: %d\nOpps deactivated: %d\nDuration: %s",
		stats.recordCount, stats.newRecords, stats.changedRecords, stats.disappearedRecords,
		stats.oppsUpserted, stats.oppsDeactivated,
		duration.Round(time.Second)))

	return nil
}

type snapshotStats struct {
	recordCount        int
	newRecords         int
	changedRecords     int
	disappearedRecords int
	reappearedRecords  int
	oppsUpserted       int
	oppsDeactivated    int
}

func (s *Service) runPipeline(ctx context.Context, runID uuid.UUID, snapshotDate time.Time, stats *snapshotStats) error {
	var result *samgov.CSVParseResult
	var downloadURL string

	if s.cfg.S3ActiveCSVKey != "" {
		// Read CSV from S3 (triggered by bulk CSV job)
		downloadURL = fmt.Sprintf("s3://%s/%s", s.cfg.S3Bucket, s.cfg.S3ActiveCSVKey)
		s.logger.Info("reading CSV from S3", "bucket", s.cfg.S3Bucket, "key", s.cfg.S3ActiveCSVKey)

		out, err := s.s3Client.GetObject(ctx, &s3.GetObjectInput{
			Bucket: aws.String(s.cfg.S3Bucket),
			Key:    aws.String(s.cfg.S3ActiveCSVKey),
		})
		if err != nil {
			return fmt.Errorf("get S3 object: %w", err)
		}
		defer out.Body.Close()

		gz, err := gzip.NewReader(out.Body)
		if err != nil {
			return fmt.Errorf("open gzip stream: %w", err)
		}
		defer gz.Close()

		result, err = samgov.ParseCSVFromReader(gz, s.cfg.RecordLimit, s.logger)
		if err != nil {
			return fmt.Errorf("parse CSV from S3: %w", err)
		}
	} else {
		// Direct download from SAM.gov (local dev / backwards compat)
		downloadURL = samgov.FullCSVURL
		var err error
		result, err = s.csv.FetchOpportunities(ctx, s.cfg.RecordLimit)
		if err != nil {
			return fmt.Errorf("fetch CSV: %w", err)
		}
	}

	// Create download log entry
	downloadEntry := &database.CSVDownloadEntry{
		RunID:        runID,
		URL:          downloadURL,
		Status:       "downloading",
		ETag:         result.ETag,
		LastModified: result.LastModified,
	}
	downloadID, err := s.db.CreateCSVDownloadEntry(ctx, downloadEntry)
	if err != nil {
		return fmt.Errorf("create download entry: %w", err)
	}

	// Convert rows to snapshot format
	s.logger.Info("converting CSV rows to snapshot format", "rows", result.ParsedRows)
	snapRows := make([]database.SnapCSVRow, 0, len(result.Rows))
	for _, raw := range result.Rows {
		snapRow := extractTypedFields(raw)
		snapRow.ContentHash = database.ComputeContentHash(raw)
		snapRows = append(snapRows, snapRow)
	}
	stats.recordCount = len(snapRows)

	// Bulk insert via COPY protocol
	s.logger.Info("bulk inserting snapshot rows", "count", len(snapRows))
	inserted, err := s.db.BulkInsertSnapCSV(ctx, runID, snapshotDate, downloadID, snapRows)
	if err != nil {
		s.db.FailCSVDownloadEntry(ctx, downloadID, err.Error())
		return fmt.Errorf("bulk insert: %w", err)
	}
	s.logger.Info("bulk insert complete", "inserted", inserted)

	// Complete download log
	s.db.CompleteCSVDownloadEntry(ctx, downloadID, len(snapRows), 0)

	// Get previous run for change detection
	prevRunID, _, prevErr := s.db.GetLastCompletedRun(ctx, "snapshot-csv")
	if prevErr != nil {
		s.logger.Warn("failed to get previous run", "error", prevErr)
	}

	if prevRunID == uuid.Nil {
		s.logger.Info("first run, all records treated as new", "count", len(snapRows))
		stats.newRecords = len(snapRows)

		ins, _, rErr := s.reconcileOpportunities(ctx, runID, snapshotDate, result.Rows, stats)
		if rErr != nil {
			s.logger.Error("reconciliation failed", "error", rErr)
		} else {
			s.logger.Info("reconciliation complete (first run)", "upserted", ins)
		}
		return nil
	}

	// Change detection
	s.logger.Info("detecting changes", "current_run", runID, "previous_run", prevRunID)
	newCount, changedCount, err := s.db.DetectChanges(ctx, runID, prevRunID, snapshotDate, s.logger)
	if err != nil {
		s.logger.Error("change detection failed", "error", err)
	} else {
		stats.newRecords = newCount
		stats.changedRecords = changedCount
		s.logger.Info("change detection complete", "new", newCount, "changed", changedCount)
	}

	// Disappearance detection
	disappearedCount, err := s.db.DetectDisappearances(ctx, runID, prevRunID, snapshotDate, s.logger)
	if err != nil {
		s.logger.Error("disappearance detection failed", "error", err)
	} else {
		stats.disappearedRecords = disappearedCount
		s.logger.Info("disappearance detection complete", "disappeared", disappearedCount)
	}

	// Reappearance detection
	reappearedCount, err := s.db.DetectReappearances(ctx, runID, snapshotDate)
	if err != nil {
		s.logger.Error("reappearance detection failed", "error", err)
	} else {
		stats.reappearedRecords = reappearedCount
		if reappearedCount > 0 {
			s.logger.Info("reappearance detection complete", "reappeared", reappearedCount)
		}
	}

	// Reconcile: populate opportunities table
	s.reconcileOpportunities(ctx, runID, snapshotDate, result.Rows, stats)

	// Mark disappeared records inactive
	deactivated, dErr := s.db.MarkDisappearedInactive(ctx, runID)
	if dErr != nil {
		s.logger.Error("failed to mark disappeared inactive", "error", dErr)
	} else if deactivated > 0 {
		stats.oppsDeactivated = deactivated
		s.logger.Info("marked disappeared opportunities inactive", "count", deactivated)
	}

	return nil
}

func (s *Service) reconcileOpportunities(ctx context.Context, runID uuid.UUID, snapshotDate time.Time, rows []map[string]string, stats *snapshotStats) (inserted, updated int, err error) {
	opps := make([]reconcile.Opportunity, 0, len(rows))
	for _, raw := range rows {
		opps = append(opps, reconcile.FromCSV(raw))
	}

	s.logger.Info("upserting opportunities", "count", len(opps))
	inserted, updated, err = s.db.UpsertOpportunities(ctx, runID, snapshotDate, opps)
	if err != nil {
		s.logger.Error("upsert opportunities failed", "error", err)
		return 0, 0, err
	}
	stats.oppsUpserted = inserted + updated
	s.logger.Info("opportunities upsert complete", "upserted", inserted+updated)
	return inserted, updated, nil
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
