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
	Store      database.Store
	Connect    func(context.Context) (database.Store, error)
	S3         bulkcsv.S3Client
	Validators bulkcsv.ValidatorObjects
	Cfg        *config.Config
	Logger     *slog.Logger
}

func (h *Handler) store(ctx context.Context) (database.Store, error) {
	if h.Store == nil && h.Connect != nil {
		s, err := h.Connect(ctx)
		if err != nil {
			return nil, err
		}
		h.Store = s
	}
	return h.Store, nil
}

// unchangedSinceLastDownload lets ~35 of the ~36 daily polls finish without
// opening a Neon connection, so the compute can scale to zero between them.
// It skips bulkcsv's "archived S3 file still exists" check; a missing archive
// is re-fetched on the next real upstream change instead.
func (h *Handler) unchangedSinceLastDownload(ctx context.Context) bool {
	if h.Validators == nil || len(bulkcsv.Sources) == 0 {
		return false
	}
	for _, src := range bulkcsv.Sources {
		v, ok := bulkcsv.LoadValidators(ctx, h.Validators, h.Cfg.S3Bucket, src.Key)
		if !ok || !bulkcsv.NotModified(ctx, src, v) {
			return false
		}
	}
	return true
}

func (h *Handler) mirrorValidators(ctx context.Context, store database.Store) {
	if h.Validators == nil {
		return
	}
	for _, src := range bulkcsv.Sources {
		etag, lastModified, err := store.GetLatestBulkCSVHeaders(ctx, src.Key)
		if err != nil || (etag == "" && lastModified == "") {
			continue
		}
		if err := bulkcsv.SaveValidators(ctx, h.Validators, h.Cfg.S3Bucket, src.Key, bulkcsv.Validators{ETag: etag, LastModified: lastModified}); err != nil {
			h.Logger.Warn("failed to mirror CSV validators to S3", "source", src.Key, "error", err)
		}
	}
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

	if h.unchangedSinceLastDownload(ctx) {
		h.Logger.Info("no new files (304 against S3-mirrored validators), skipped DB", "execution_id", executionID.String())
		return &Output{ExecutionID: executionID.String()}, nil
	}

	store, err := h.store(ctx)
	if err != nil {
		return nil, fmt.Errorf("connect database: %w", err)
	}

	// Idempotency: if this execution already completed, return cached output
	existing, err := store.GetPipelineRun(ctx, executionID)
	if err != nil {
		h.Logger.Warn("failed to check existing pipeline run", "error", err)
	}
	if existing != nil && existing.Status == "completed" {
		h.Logger.Info("pipeline run already completed, returning cached result", "id", executionID)
		return h.buildCachedOutput(existing), nil
	}

	results, err := bulkcsv.Run(ctx, h.Cfg, store, h.S3, h.Logger)
	if err != nil {
		return nil, fmt.Errorf("bulk csv run: %w", err)
	}
	h.mirrorValidators(ctx, store)

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
		stepID, stepErr := store.CreatePipelineStep(ctx, executionID, "download-csvs")
		if stepErr != nil {
			h.Logger.Warn("failed to create pipeline step", "error", stepErr)
		}

		dur := int(time.Since(start).Milliseconds())
		stats := map[string]any{
			"summary":   bulkcsv.FormatSummary(results),
			"new_files": len(files),
		}

		if stepErr == nil {
			if err := store.CompletePipelineStep(ctx, stepID, stats, dur); err != nil {
				h.Logger.Warn("failed to complete pipeline step", "error", err)
			}
		}

		// Backward compat: still create pipeline_run for ingestion_run FK
		runID, err := store.CreatePipelineRun(ctx, executionID, "download-csvs", nil)
		if err != nil {
			h.Logger.Warn("failed to create pipeline run (compat)", "error", err)
		} else {
			if err := store.CompletePipelineRun(ctx, runID, stats, dur); err != nil {
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
	h := &Handler{
		Connect: func(ctx context.Context) (database.Store, error) {
			return database.New(ctx, cfg.DatabaseURL)
		},
		S3:         s3Client,
		Validators: s3Client,
		Cfg:        &cfg,
		Logger:     logger,
	}
	lambda.Start(h.Handle)
}
