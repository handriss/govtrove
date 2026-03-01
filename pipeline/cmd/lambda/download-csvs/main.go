package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"strings"
	"time"

	"github.com/aws/aws-lambda-go/lambda"
	awsconfig "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/secretsmanager"
	"github.com/getsentry/sentry-go"
	"github.com/google/uuid"
	"github.com/handriss/govtrove/pipeline/internal/bulkcsv"
	"github.com/handriss/govtrove/pipeline/internal/config"
	"github.com/handriss/govtrove/pipeline/internal/database"
)

const (
	envDatabaseURLSecretARN = "DATABASE_URL_SECRET_ARN"
	envS3Bucket             = "S3_BUCKET"
	envAWSRegion            = "AWS_REGION_NAME"
	envSentryDSN            = "SENTRY_DSN" // optional
)

var (
	db       *database.DB
	s3Client *s3.Client
	cfg      config.Config
	logger   *slog.Logger
)

func init() {
	ctx := context.Background()

	logger = slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo}))
	slog.SetDefault(logger)

	secretARN := os.Getenv(envDatabaseURLSecretARN)
	if secretARN == "" {
		return
	}

	if err := config.RequireEnv(envDatabaseURLSecretARN, envS3Bucket, envAWSRegion); err != nil {
		logger.Error("config validation failed", "error", err)
		os.Exit(1)
	}

	if dsn := os.Getenv(envSentryDSN); dsn != "" {
		sentry.Init(sentry.ClientOptions{
			Dsn:              dsn,
			Environment:      "production",
			AttachStacktrace: true,
		})
	}

	cfg.S3Bucket = os.Getenv(envS3Bucket)
	cfg.AWSRegion = os.Getenv(envAWSRegion)
	cfg.S3ArchiveEnabled = true

	awsCfg, err := awsconfig.LoadDefaultConfig(ctx, awsconfig.WithRegion(cfg.AWSRegion))
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

	cfg.DatabaseURL = *result.SecretString

	db, err = database.New(ctx, cfg.DatabaseURL)
	if err != nil {
		logger.Error("failed to connect to database", "error", err)
		os.Exit(1)
	}

	s3Client = s3.NewFromConfig(awsCfg)
	logger.Info("cold start complete", "bucket", cfg.S3Bucket)
}

type Input struct {
	ID string `json:"id"`
}

type File struct {
	Type   string `json:"type"`
	S3Key  string `json:"s3_key"`
	Source string `json:"source"`
}

type Output struct {
	ExecutionID string `json:"execution_id"`
	Files       []File `json:"files"`
}

// Handler holds dependencies for the download-csvs Lambda.
type Handler struct {
	Store  database.Store
	S3     bulkcsv.S3Client
	Cfg    *config.Config
	Logger *slog.Logger
}

func (h *Handler) Handle(ctx context.Context, event json.RawMessage) (_ *Output, retErr error) {
	defer func() {
		if retErr != nil {
			sentry.CaptureException(retErr)
		}
		sentry.Flush(2 * time.Second)
	}()
	start := time.Now()

	// Parse execution ID from Step Functions input
	var input Input
	if err := json.Unmarshal(event, &input); err != nil {
		h.Logger.Warn("failed to parse input, will generate run ID", "error", err)
	}

	var executionID uuid.UUID
	if input.ID != "" {
		parsed, err := uuid.Parse(input.ID)
		if err != nil {
			h.Logger.Warn("invalid UUID in input, will generate run ID", "id", input.ID, "error", err)
		} else {
			executionID = parsed
		}
	}
	if executionID == uuid.Nil {
		executionID = uuid.New()
	}

	// Idempotency: if this execution already completed, return cached output
	existing, err := h.Store.GetPipelineRun(ctx, executionID)
	if err != nil {
		h.Logger.Warn("failed to check existing pipeline run", "error", err)
	}
	if existing != nil && existing.Status == "completed" {
		h.Logger.Info("pipeline run already completed, returning cached result", "id", executionID)
		return h.buildCachedOutput(existing), nil
	}

	results, err := bulkcsv.Run(ctx, h.Cfg, h.Store, h.S3, h.Logger)
	if err != nil {
		return nil, fmt.Errorf("bulk csv run: %w", err)
	}

	var failed []string
	for _, r := range results {
		if r.Outcome == "error" {
			failed = append(failed, r.Source)
		}
	}
	if len(failed) > 0 {
		errMsg := fmt.Sprintf("sources failed: %s", strings.Join(failed, ", "))
		return nil, fmt.Errorf("%s", errMsg)
	}

	var files []File
	for _, r := range results {
		if r.Outcome != "new_file" {
			continue
		}
		files = append(files, File{
			Type:   string(r.Type),
			S3Key:  r.S3Key,
			Source: r.Source,
		})
	}

	// Only write DB rows when there are new files to process
	if len(files) > 0 {
		// Pipeline step tracking
		stepID, stepErr := h.Store.CreatePipelineStep(ctx, executionID, "download-csvs")
		if stepErr != nil {
			h.Logger.Warn("failed to create pipeline step", "error", stepErr)
		}

		dur := int(time.Since(start).Milliseconds())
		stats := map[string]any{
			"summary":   bulkcsv.FormatSummary(results),
			"new_files": len(files),
		}

		if stepErr == nil {
			if err := h.Store.CompletePipelineStep(ctx, stepID, stats, dur); err != nil {
				h.Logger.Warn("failed to complete pipeline step", "error", err)
			}
		}

		// Backward compat: still create pipeline_run for ingestion_run FK
		runID, err := h.Store.CreatePipelineRun(ctx, executionID, "download-csvs", nil)
		if err != nil {
			h.Logger.Warn("failed to create pipeline run (compat)", "error", err)
		} else {
			if err := h.Store.CompletePipelineRun(ctx, runID, stats, dur); err != nil {
				h.Logger.Warn("failed to complete pipeline run (compat)", "error", err)
			}
		}

		h.Logger.Info("handler complete",
			"execution_id", executionID.String(),
			"new_files", len(files),
			"duration_ms", dur,
		)
	} else {
		h.Logger.Info("no new files, skipping DB writes", "execution_id", executionID.String())
	}

	return &Output{
		ExecutionID: executionID.String(),
		Files:       files,
	}, nil
}

func (h *Handler) buildCachedOutput(run *database.PipelineRun) *Output {
	out := &Output{ExecutionID: run.ID.String()}
	if run.Stats != nil {
		if filesRaw, ok := run.Stats["files"]; ok {
			if filesJSON, err := json.Marshal(filesRaw); err == nil {
				json.Unmarshal(filesJSON, &out.Files)
			}
		}
	}
	return out
}

func main() {
	h := &Handler{Store: db, S3: s3Client, Cfg: &cfg, Logger: logger}
	lambda.Start(h.Handle)
}
