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
	"github.com/handriss/govtrove/pipeline/internal/database"
	"github.com/handriss/govtrove/pipeline/internal/ingest"
	"github.com/handriss/govtrove/pipeline/internal/samgov"
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

	secretARN := os.Getenv("DATABASE_URL_SECRET_ARN")
	if secretARN == "" {
		return
	}

	if dsn := os.Getenv("SENTRY_DSN"); dsn != "" {
		sentry.Init(sentry.ClientOptions{
			Dsn:              dsn,
			Environment:      "production",
			AttachStacktrace: true,
		})
	}

	bucket = os.Getenv("S3_BUCKET")
	if bucket == "" {
		bucket = "govtrove-data"
	}

	region := os.Getenv("AWS_REGION_NAME")
	if region == "" {
		region = "us-east-1"
	}

	awsCfg, err := awsconfig.LoadDefaultConfig(ctx, awsconfig.WithRegion(region))
	if err != nil {
		logger.Error("failed to load AWS config", "error", err)
		os.Exit(1)
	}

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

const jobType = "ingest-archived"

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
		}
		sentry.Flush(2 * time.Second)
	}()
	var input Input
	if err := json.Unmarshal(event, &input); err != nil {
		return nil, fmt.Errorf("unmarshal input: %w", err)
	}
	h.Logger.Info("starting ingest-archived", "s3_key", input.File.S3Key)

	start := time.Now()
	snapshotDate := time.Now().UTC()

	runID, err := h.Store.CreateIngestionRun(ctx, jobType)
	if err != nil {
		return nil, fmt.Errorf("create ingestion run: %w", err)
	}

	recordCount, err := h.runPipeline(ctx, runID, snapshotDate, input.File.S3Key)
	durationMs := int(time.Since(start).Milliseconds())

	if err != nil {
		h.Store.FailIngestionRun(ctx, runID, err.Error(), durationMs)
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
	downloadURL := fmt.Sprintf("s3://%s/%s", h.Bucket, s3Key)

	out, err := h.S3.GetObject(ctx, &s3.GetObjectInput{
		Bucket: aws.String(h.Bucket),
		Key:    aws.String(s3Key),
	})
	if err != nil {
		return 0, fmt.Errorf("get S3 object: %w", err)
	}
	defer out.Body.Close()

	return h.processStream(ctx, runID, snapshotDate, downloadURL, out.Body)
}

func (h *Handler) processStream(ctx context.Context, runID uuid.UUID, snapshotDate time.Time, downloadURL string, body io.ReadCloser) (int, error) {
	gz, err := gzip.NewReader(body)
	if err != nil {
		return 0, fmt.Errorf("open gzip stream: %w", err)
	}
	defer gz.Close()

	result, err := samgov.ParseCSVFromReader(gz, 0, h.Logger)
	if err != nil {
		return 0, fmt.Errorf("parse CSV: %w", err)
	}

	downloadID, err := h.Store.CreateCSVDownloadEntry(ctx, &database.CSVDownloadEntry{
		RunID:  runID,
		URL:    downloadURL,
		Status: "downloading",
	})
	if err != nil {
		return 0, fmt.Errorf("create download entry: %w", err)
	}

	snapRows := make([]database.SnapCSVRow, 0, len(result.Rows))
	for _, raw := range result.Rows {
		snapRows = append(snapRows, ingest.ExtractSnapCSVRow(raw))
	}
	recordCount := len(snapRows)

	h.Logger.Info("bulk inserting snapshot rows", "count", recordCount)
	if _, err := h.Store.BulkInsertSnapCSV(ctx, runID, snapshotDate, downloadID, snapRows); err != nil {
		h.Store.FailCSVDownloadEntry(ctx, downloadID, err.Error())
		return recordCount, fmt.Errorf("bulk insert: %w", err)
	}

	h.Store.CompleteCSVDownloadEntry(ctx, downloadID, recordCount, 0)

	return recordCount, nil
}

func main() {
	h := &Handler{Store: db, S3: s3Client, Bucket: bucket, Logger: logger}
	lambda.Start(h.Handle)
}
