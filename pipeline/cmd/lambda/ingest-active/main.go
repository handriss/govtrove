package main

import (
	"compress/gzip"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"os"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	awsconfig "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/secretsmanager"
	"github.com/aws/aws-lambda-go/lambda"
	"github.com/getsentry/sentry-go"
	"github.com/google/uuid"
	"github.com/handriss/govtrove/pipeline/internal/config"
	"github.com/handriss/govtrove/pipeline/internal/database"
	"github.com/handriss/govtrove/pipeline/internal/ingest"
	"github.com/handriss/govtrove/pipeline/internal/samgov"
)

const (
	envDatabaseURLSecretARN = "DATABASE_URL_SECRET_ARN"
	envS3Bucket             = "S3_BUCKET"
	envAWSRegion            = "AWS_REGION_NAME"
	envSentryDSN            = "SENTRY_DSN"
)

var (
	db       *database.DB
	s3Client *s3.Client
	bucket   string
	logger   *slog.Logger
)

func init() {
	ctx := context.Background()

	logger = slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo}))
	slog.SetDefault(logger)

	if os.Getenv(envDatabaseURLSecretARN) == "" {
		return
	}

	if err := config.RequireEnv(envDatabaseURLSecretARN, envS3Bucket, envAWSRegion); err != nil {
		logger.Error("missing required env vars", "error", err)
		os.Exit(1)
	}

	if dsn := os.Getenv(envSentryDSN); dsn != "" {
		sentry.Init(sentry.ClientOptions{
			Dsn:              dsn,
			Environment:      "production",
			AttachStacktrace: true,
		})
	}

	bucket = os.Getenv(envS3Bucket)
	region := os.Getenv(envAWSRegion)

	awsCfg, err := awsconfig.LoadDefaultConfig(ctx, awsconfig.WithRegion(region))
	if err != nil {
		logger.Error("failed to load AWS config", "error", err)
		os.Exit(1)
	}

	secretARN := os.Getenv(envDatabaseURLSecretARN)
	smClient := secretsmanager.NewFromConfig(awsCfg)
	result, err := smClient.GetSecretValue(ctx, &secretsmanager.GetSecretValueInput{
		SecretId: &secretARN,
	})
	if err != nil {
		logger.Error("failed to get database URL from Secrets Manager", "error", err)
		os.Exit(1)
	}

	db, err = database.New(ctx, *result.SecretString)
	if err != nil {
		logger.Error("failed to connect to database", "error", err)
		os.Exit(1)
	}

	s3Client = s3.NewFromConfig(awsCfg)
	logger.Info("cold start complete", "bucket", bucket)
}

type Input struct {
	ExecutionID string `json:"execution_id"`
	File        struct {
		Type   string `json:"type"`
		S3Key  string `json:"s3_key"`
		Source string `json:"source"`
	} `json:"file"`
}

type Output struct {
	Status  string `json:"status"`
	RunID   string `json:"run_id"`
	JobType string `json:"job_type"`
}

const (
	jobType   = "snapshot-csv"
	batchSize = 5000
)

// S3Getter abstracts the S3 GetObject call for testability.
type S3Getter interface {
	GetObject(ctx context.Context, params *s3.GetObjectInput, optFns ...func(*s3.Options)) (*s3.GetObjectOutput, error)
}

// Handler holds dependencies for the ingest-active Lambda.
type Handler struct {
	Store  database.Store
	S3     S3Getter
	Bucket string
	Logger *slog.Logger
}

func (h *Handler) Handle(ctx context.Context, event json.RawMessage) (_ *Output, retErr error) {
	defer func() {
		if retErr != nil {
			sentry.CaptureException(retErr)
			sentry.Flush(500 * time.Millisecond)
		}
	}()

	var input Input
	if err := json.Unmarshal(event, &input); err != nil {
		return nil, fmt.Errorf("unmarshal input: %w", err)
	}
	h.Logger.Info("starting ingest-active", "s3_key", input.File.S3Key)

	var executionID *uuid.UUID
	if input.ExecutionID != "" {
		parsed, err := uuid.Parse(input.ExecutionID)
		if err == nil {
			executionID = &parsed
		}
	}

	// Pipeline step tracking
	var stepID uuid.UUID
	if executionID != nil {
		sid, err := h.Store.CreatePipelineStep(ctx, *executionID, "ingest-active")
		if err != nil {
			h.Logger.Warn("failed to create pipeline step", "error", err)
		} else {
			stepID = sid
		}
	}

	start := time.Now()
	snapshotDate := time.Now().UTC()

	runID, err := h.Store.CreateIngestionRun(ctx, jobType, executionID)
	if err != nil {
		return nil, fmt.Errorf("create ingestion run: %w", err)
	}

	stats, err := h.runPipeline(ctx, runID, snapshotDate, input.File.S3Key)
	durationMs := int(time.Since(start).Milliseconds())

	if err != nil {
		if failErr := h.Store.FailIngestionRun(ctx, runID, err.Error(), durationMs); failErr != nil {
			h.Logger.Error("failed to mark ingestion run as failed", "error", failErr)
		}
		if stepID != uuid.Nil {
			_ = h.Store.FailPipelineStep(ctx, stepID, err.Error(), durationMs)
		}
		return nil, fmt.Errorf("ingest-active failed: %w", err)
	}

	if dbErr := h.Store.CompleteIngestionRun(ctx, runID, database.RunStats{
		Fetched:    stats.recordCount,
		DurationMs: durationMs,
	}); dbErr != nil {
		h.Logger.Error("failed to complete ingestion run", "error", dbErr)
	}

	if stepID != uuid.Nil {
		stepStats := map[string]any{
			"records":      stats.recordCount,
			"new":          stats.newRecords,
			"changed":      stats.changedRecords,
			"disappeared":  stats.disappearedRecords,
		}
		// Use a detached context — the Lambda context may be nearly expired
		// after processing a large CSV, causing this write to fail silently.
		completionCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if err := h.Store.CompletePipelineStep(completionCtx, stepID, stepStats, durationMs); err != nil {
			h.Logger.Warn("failed to complete pipeline step", "error", err)
		}
	}

	h.Logger.Info("ingest-active complete",
		"run_id", runID,
		"records", stats.recordCount,
		"new", stats.newRecords,
		"changed", stats.changedRecords,
		"disappeared", stats.disappearedRecords,
		"duration_ms", durationMs,
	)

	return &Output{Status: "ok", RunID: runID.String(), JobType: jobType}, nil
}

type pipelineStats struct {
	recordCount        int
	newRecords         int
	changedRecords     int
	disappearedRecords int
}

func (h *Handler) runPipeline(ctx context.Context, runID uuid.UUID, snapshotDate time.Time, s3Key string) (*pipelineStats, error) {
	// Idempotency: check if this file was already ingested
	bulkLog, err := h.Store.GetBulkCSVLogByS3Key(ctx, s3Key)
	if err != nil {
		return nil, fmt.Errorf("lookup bulk_csv_log: %w", err)
	}
	if bulkLog == nil {
		return nil, fmt.Errorf("no bulk_csv_log record for s3_key %q — download-csvs should have created it", s3Key)
	}
	if bulkLog.Status != nil && *bulkLog.Status == "completed" {
		h.Logger.Info("file already ingested, skipping", "s3_key", s3Key, "bulk_csv_log_id", bulkLog.ID)
		rc := 0
		if bulkLog.RowCount != nil {
			rc = *bulkLog.RowCount
		}
		return &pipelineStats{recordCount: rc}, nil
	}

	downloadID := int64(bulkLog.ID)

	// Mark as ingesting
	if err := h.Store.UpdateBulkCSVLogIngestion(ctx, bulkLog.ID, runID, 0, "ingesting"); err != nil {
		h.Logger.Error("failed to update bulk_csv_log to ingesting", "error", err)
	}

	out, err := h.S3.GetObject(ctx, &s3.GetObjectInput{
		Bucket: aws.String(h.Bucket),
		Key:    aws.String(s3Key),
	})
	if err != nil {
		if updateErr := h.Store.UpdateBulkCSVLogIngestion(ctx, bulkLog.ID, runID, 0, "failed"); updateErr != nil {
			h.Logger.Error("failed to update bulk_csv_log to failed", "error", updateErr)
		}
		return nil, fmt.Errorf("get S3 object: %w", err)
	}
	defer out.Body.Close()

	stats, err := h.processStream(ctx, runID, snapshotDate, downloadID, bulkLog.ID, out.Body)
	if err != nil {
		partialCount := 0
		if stats != nil {
			partialCount = stats.recordCount
		}
		if updateErr := h.Store.UpdateBulkCSVLogIngestion(ctx, bulkLog.ID, runID, partialCount, "failed"); updateErr != nil {
			h.Logger.Error("failed to update bulk_csv_log to failed", "error", updateErr)
		}
		return stats, err
	}

	if err := h.Store.UpdateBulkCSVLogIngestion(ctx, bulkLog.ID, runID, stats.recordCount, "completed"); err != nil {
		h.Logger.Error("failed to update bulk_csv_log to completed", "error", err)
	}

	return stats, nil
}

func (h *Handler) processStream(ctx context.Context, runID uuid.UUID, snapshotDate time.Time, downloadID int64, bulkLogID int, body io.ReadCloser) (*pipelineStats, error) {
	gz, err := gzip.NewReader(body)
	if err != nil {
		return nil, fmt.Errorf("open gzip stream: %w", err)
	}
	defer gz.Close()

	batch := make([]database.SnapCSVRow, 0, batchSize)
	stats := &pipelineStats{}

	flush := func() error {
		if len(batch) == 0 {
			return nil
		}
		h.Logger.Info("bulk inserting snapshot rows", "count", len(batch), "total_so_far", stats.recordCount)
		if _, err := h.Store.BulkInsertSnapCSV(ctx, runID, snapshotDate, downloadID, batch); err != nil {
			return fmt.Errorf("bulk insert: %w", err)
		}
		batch = batch[:0]
		return nil
	}

	_, parsed, _, parseErr := samgov.ParseCSVStream(gz, h.Logger, func(row map[string]string) error {
		batch = append(batch, ingest.ExtractSnapCSVRow(row))
		stats.recordCount++
		if len(batch) >= batchSize {
			if err := flush(); err != nil {
				return err
			}
		}
		return nil
	})
	if parseErr != nil {
		return stats, fmt.Errorf("parse CSV: %w", parseErr)
	}

	// Flush remaining rows
	if err := flush(); err != nil {
		return stats, err
	}

	// recordCount is tracked via callback; sanity check against parsed count
	if stats.recordCount != parsed {
		h.Logger.Warn("record count mismatch", "tracked", stats.recordCount, "parsed", parsed)
	}

	// Change detection
	prevRunID, _, prevErr := h.Store.GetLastCompletedRun(ctx, jobType)
	if prevErr != nil {
		h.Logger.Warn("failed to get previous run", "error", prevErr)
	}

	if prevRunID != uuid.Nil {
		newCount, changedCount, err := h.Store.DetectChanges(ctx, runID, prevRunID, snapshotDate, h.Logger)
		if err != nil {
			h.Logger.Error("change detection failed", "error", err)
		} else {
			stats.newRecords = newCount
			stats.changedRecords = changedCount
		}

		disappearedCount, err := h.Store.DetectDisappearances(ctx, runID, prevRunID, snapshotDate, h.Logger)
		if err != nil {
			h.Logger.Error("disappearance detection failed", "error", err)
		} else {
			stats.disappearedRecords = disappearedCount
		}

		reappearedCount, err := h.Store.DetectReappearances(ctx, runID, snapshotDate)
		if err != nil {
			h.Logger.Error("reappearance detection failed", "error", err)
		} else if reappearedCount > 0 {
			h.Logger.Info("reappearances detected", "count", reappearedCount)
		}
	} else {
		h.Logger.Info("first run — skipping change detection", "records", stats.recordCount)
		stats.newRecords = stats.recordCount
	}

	return stats, nil
}

func main() {
	h := &Handler{Store: db, S3: s3Client, Bucket: bucket, Logger: logger}
	lambda.Start(h.Handle)
}
