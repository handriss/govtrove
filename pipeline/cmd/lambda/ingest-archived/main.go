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
	PipelineRunID string `json:"pipeline_run_id"`
	File          struct {
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
	jobType   = "ingest-archived"
	batchSize = 5000
)

// S3Getter abstracts the S3 GetObject call for testability.
type S3Getter interface {
	GetObject(ctx context.Context, params *s3.GetObjectInput, optFns ...func(*s3.Options)) (*s3.GetObjectOutput, error)
}

// Handler holds dependencies for the ingest-archived Lambda.
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
	h.Logger.Info("starting ingest-archived", "s3_key", input.File.S3Key)

	var pipelineRunID *uuid.UUID
	if input.PipelineRunID != "" {
		parsed, err := uuid.Parse(input.PipelineRunID)
		if err == nil {
			pipelineRunID = &parsed
		}
	}

	start := time.Now()
	snapshotDate := time.Now().UTC()

	runID, err := h.Store.CreateIngestionRun(ctx, jobType, pipelineRunID)
	if err != nil {
		return nil, fmt.Errorf("create ingestion run: %w", err)
	}

	recordCount, err := h.runPipeline(ctx, runID, snapshotDate, input.File.S3Key)
	durationMs := int(time.Since(start).Milliseconds())

	if err != nil {
		if failErr := h.Store.FailIngestionRun(ctx, runID, err.Error(), durationMs); failErr != nil {
			h.Logger.Error("failed to mark ingestion run as failed", "error", failErr)
		}
		return nil, fmt.Errorf("ingest-archived failed: %w", err)
	}

	if dbErr := h.Store.CompleteIngestionRun(ctx, runID, database.RunStats{
		Fetched:    recordCount,
		DurationMs: durationMs,
	}); dbErr != nil {
		h.Logger.Error("failed to complete ingestion run", "error", dbErr)
	}

	h.Logger.Info("ingest-archived complete",
		"run_id", runID,
		"records", recordCount,
		"duration_ms", durationMs,
	)

	return &Output{Status: "ok", RunID: runID.String(), JobType: jobType}, nil
}

func (h *Handler) runPipeline(ctx context.Context, runID uuid.UUID, snapshotDate time.Time, s3Key string) (int, error) {
	// Idempotency: check if this file was already ingested
	bulkLog, err := h.Store.GetBulkCSVLogByS3Key(ctx, s3Key)
	if err != nil {
		return 0, fmt.Errorf("lookup bulk_csv_log: %w", err)
	}
	if bulkLog == nil {
		return 0, fmt.Errorf("no bulk_csv_log record for s3_key %q — download-csvs should have created it", s3Key)
	}
	if bulkLog.Status != nil && *bulkLog.Status == "completed" {
		h.Logger.Info("file already ingested, skipping", "s3_key", s3Key, "bulk_csv_log_id", bulkLog.ID)
		rc := 0
		if bulkLog.RowCount != nil {
			rc = *bulkLog.RowCount
		}
		return rc, nil
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
		return 0, fmt.Errorf("get S3 object: %w", err)
	}
	defer out.Body.Close()

	recordCount, err := h.processStream(ctx, runID, snapshotDate, downloadID, out.Body)
	if err != nil {
		if updateErr := h.Store.UpdateBulkCSVLogIngestion(ctx, bulkLog.ID, runID, recordCount, "failed"); updateErr != nil {
			h.Logger.Error("failed to update bulk_csv_log to failed", "error", updateErr)
		}
		return recordCount, err
	}

	if err := h.Store.UpdateBulkCSVLogIngestion(ctx, bulkLog.ID, runID, recordCount, "completed"); err != nil {
		h.Logger.Error("failed to update bulk_csv_log to completed", "error", err)
	}

	return recordCount, nil
}

func (h *Handler) processStream(ctx context.Context, runID uuid.UUID, snapshotDate time.Time, downloadID int64, body io.ReadCloser) (int, error) {
	gz, err := gzip.NewReader(body)
	if err != nil {
		return 0, fmt.Errorf("open gzip stream: %w", err)
	}
	defer gz.Close()

	batch := make([]database.SnapCSVRow, 0, batchSize)
	recordCount := 0

	flush := func() error {
		if len(batch) == 0 {
			return nil
		}
		h.Logger.Info("bulk inserting snapshot rows", "count", len(batch), "total_so_far", recordCount)
		if _, err := h.Store.BulkInsertSnapCSV(ctx, runID, snapshotDate, downloadID, batch); err != nil {
			return fmt.Errorf("bulk insert: %w", err)
		}
		batch = batch[:0]
		return nil
	}

	_, parsed, _, parseErr := samgov.ParseCSVStream(gz, h.Logger, func(row map[string]string) error {
		batch = append(batch, ingest.ExtractSnapCSVRow(row))
		recordCount++
		if len(batch) >= batchSize {
			if err := flush(); err != nil {
				return err
			}
		}
		return nil
	})
	if parseErr != nil {
		return recordCount, fmt.Errorf("parse CSV: %w", parseErr)
	}

	// Flush remaining rows
	if err := flush(); err != nil {
		return recordCount, err
	}

	if recordCount != parsed {
		h.Logger.Warn("record count mismatch", "tracked", recordCount, "parsed", parsed)
	}

	return recordCount, nil
}

func main() {
	h := &Handler{Store: db, S3: s3Client, Bucket: bucket, Logger: logger}
	lambda.Start(h.Handle)
}
