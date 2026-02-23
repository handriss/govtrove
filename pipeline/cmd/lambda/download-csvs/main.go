package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"strings"
	"time"

	awsconfig "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/secretsmanager"
	"github.com/aws/aws-lambda-go/lambda"
	"github.com/getsentry/sentry-go"
	"github.com/handriss/govtrove/pipeline/internal/bulkcsv"
	"github.com/handriss/govtrove/pipeline/internal/config"
	"github.com/handriss/govtrove/pipeline/internal/database"
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

	cfg.S3Bucket = os.Getenv("S3_BUCKET")
	if cfg.S3Bucket == "" {
		cfg.S3Bucket = "govtrove-data"
	}

	cfg.AWSRegion = os.Getenv("AWS_REGION_NAME")
	if cfg.AWSRegion == "" {
		cfg.AWSRegion = "us-east-1"
	}

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

type File struct {
	Type   string `json:"type"`
	S3Key  string `json:"s3_key"`
	Source string `json:"source"`
}

type Output struct {
	PipelineRunID string `json:"pipeline_run_id"`
	Files         []File `json:"files"`
}

// Handler holds dependencies for the download-csvs Lambda.
type Handler struct {
	Store    database.Store
	S3       bulkcsv.S3Client
	Cfg      *config.Config
	Logger   *slog.Logger
}

func (h *Handler) Handle(ctx context.Context, event json.RawMessage) (_ *Output, retErr error) {
	defer func() {
		if retErr != nil {
			sentry.CaptureException(retErr)
		}
		sentry.Flush(2 * time.Second)
	}()
	start := time.Now()

	runID, err := h.Store.CreatePipelineRun(ctx, "download-csvs", nil)
	if err != nil {
		return nil, fmt.Errorf("create pipeline run: %w", err)
	}

	results, err := bulkcsv.Run(ctx, h.Cfg, h.Store, h.S3, h.Logger)
	if err != nil {
		dur := int(time.Since(start).Milliseconds())
		_ = h.Store.FailPipelineRun(ctx, runID, err.Error(), dur)
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
		dur := int(time.Since(start).Milliseconds())
		_ = h.Store.FailPipelineRun(ctx, runID, errMsg, dur)
		return nil, fmt.Errorf("%s", errMsg)
	}

	var files []File
	for _, r := range results {
		if r.Outcome != "new_file" {
			continue
		}
		f := File{
			S3Key:  r.S3Key,
			Source: r.Source,
		}
		if r.Source == "active" {
			f.Type = "active"
		} else if strings.Contains(r.Source, "archived") {
			f.Type = "archived"
		}
		files = append(files, f)
	}

	dur := int(time.Since(start).Milliseconds())
	stats := map[string]any{
		"summary":   bulkcsv.FormatSummary(results),
		"new_files": len(files),
	}
	if err := h.Store.CompletePipelineRun(ctx, runID, stats, dur); err != nil {
		h.Logger.Warn("failed to complete pipeline run", "error", err)
	}

	h.Logger.Info("handler complete",
		"pipeline_run_id", runID.String(),
		"new_files", len(files),
		"duration_ms", dur,
	)

	return &Output{
		PipelineRunID: runID.String(),
		Files:         files,
	}, nil
}

func main() {
	h := &Handler{Store: db, S3: s3Client, Cfg: &cfg, Logger: logger}
	lambda.Start(h.Handle)
}
