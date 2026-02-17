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
		logger.Error("DATABASE_URL_SECRET_ARN not set")
		os.Exit(1)
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

func handler(ctx context.Context, event json.RawMessage) (*Output, error) {
	start := time.Now()

	runID, err := db.CreatePipelineRun(ctx, "download-csvs", nil)
	if err != nil {
		return nil, fmt.Errorf("create pipeline run: %w", err)
	}

	results, err := bulkcsv.Run(ctx, &cfg, db, s3Client, logger)
	if err != nil {
		dur := int(time.Since(start).Milliseconds())
		_ = db.FailPipelineRun(ctx, runID, err.Error(), dur)
		return nil, fmt.Errorf("bulk csv run: %w", err)
	}

	// Check for errors in individual sources
	var failed []string
	for _, r := range results {
		if r.Outcome == "error" {
			failed = append(failed, r.Source)
		}
	}
	if len(failed) > 0 {
		errMsg := fmt.Sprintf("sources failed: %s", strings.Join(failed, ", "))
		dur := int(time.Since(start).Milliseconds())
		_ = db.FailPipelineRun(ctx, runID, errMsg, dur)
		return nil, fmt.Errorf("%s", errMsg)
	}

	// Build output files for Step Functions (only new files)
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
	if err := db.CompletePipelineRun(ctx, runID, stats, dur); err != nil {
		logger.Warn("failed to complete pipeline run", "error", err)
	}

	logger.Info("handler complete",
		"pipeline_run_id", runID.String(),
		"new_files", len(files),
		"duration_ms", dur,
	)

	return &Output{
		PipelineRunID: runID.String(),
		Files:         files,
	}, nil
}

func main() { lambda.Start(handler) }
