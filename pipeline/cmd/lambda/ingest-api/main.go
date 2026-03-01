package main

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"time"

	awsconfig "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/secretsmanager"
	"github.com/aws/aws-lambda-go/lambda"
	"github.com/getsentry/sentry-go"
	"github.com/google/uuid"
	"github.com/handriss/govtrove/pipeline/internal/config"
	"github.com/handriss/govtrove/pipeline/internal/database"
	"github.com/handriss/govtrove/pipeline/internal/samgov"
)

const (
	envDatabaseURLSecretARN = "DATABASE_URL_SECRET_ARN"
	envAWSRegion            = "AWS_REGION_NAME"
	envSentryDSN            = "SENTRY_DSN"
	envSAMAPIKeySecretARN   = "SAM_API_KEY_SECRET_ARN"
	envSAMAPIKey            = "SAM_API_KEY"
)

var (
	db     *database.DB
	apiKey string
	logger *slog.Logger
)

func init() {
	ctx := context.Background()

	logger = slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo}))
	slog.SetDefault(logger)

	if os.Getenv(envDatabaseURLSecretARN) == "" {
		return
	}

	if err := config.RequireEnv(envDatabaseURLSecretARN, envAWSRegion); err != nil {
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

	region := os.Getenv(envAWSRegion)
	awsCfg, err := awsconfig.LoadDefaultConfig(ctx, awsconfig.WithRegion(region))
	if err != nil {
		logger.Error("failed to load AWS config", "error", err)
		os.Exit(1)
	}

	smClient := secretsmanager.NewFromConfig(awsCfg)

	dbSecretARN := os.Getenv(envDatabaseURLSecretARN)
	dbResult, err := smClient.GetSecretValue(ctx, &secretsmanager.GetSecretValueInput{
		SecretId: &dbSecretARN,
	})
	if err != nil {
		logger.Error("failed to get database URL from Secrets Manager", "error", err)
		os.Exit(1)
	}

	db, err = database.New(ctx, *dbResult.SecretString)
	if err != nil {
		logger.Error("failed to connect to database", "error", err)
		os.Exit(1)
	}

	apiKeyARN := os.Getenv(envSAMAPIKeySecretARN)
	if apiKeyARN != "" {
		keyResult, err := smClient.GetSecretValue(ctx, &secretsmanager.GetSecretValueInput{
			SecretId: &apiKeyARN,
		})
		if err != nil {
			logger.Error("failed to get SAM API key from Secrets Manager", "error", err)
			os.Exit(1)
		}
		apiKey = *keyResult.SecretString
	} else {
		apiKey = os.Getenv(envSAMAPIKey)
	}

	if apiKey == "" {
		logger.Error("no SAM API key configured")
		os.Exit(1)
	}

	logger.Info("cold start complete")
}

type Input struct {
	PipelineRunID string `json:"pipeline_run_id"`
}

type Output struct {
	Status       string `json:"status"`
	RunID        string `json:"run_id"`
	JobType      string `json:"job_type"`
	TotalFetched int    `json:"total_fetched"`
	Upserted     int    `json:"upserted"`
}

const jobType = "snapshot-api"

type APIFetcher interface {
	FetchAll(ctx context.Context, postedFrom, postedTo string) ([]samgov.OpportunityData, []json.RawMessage, int, error)
}

type Handler struct {
	Store  database.Store
	Logger *slog.Logger
	API    APIFetcher
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

	h.Logger.Info("starting ingest-api")
	start := time.Now()
	snapshotDate := time.Now().UTC()

	runID, err := h.Store.CreateIngestionRun(ctx, jobType, nil)
	if err != nil {
		return nil, fmt.Errorf("create ingestion run: %w", err)
	}

	totalFetched, upserted, err := h.runPipeline(ctx, runID, snapshotDate)
	durationMs := int(time.Since(start).Milliseconds())

	if err != nil {
		if failErr := h.Store.FailIngestionRun(ctx, runID, err.Error(), durationMs); failErr != nil {
			h.Logger.Error("failed to mark ingestion run as failed", "error", failErr)
		}
		return nil, fmt.Errorf("ingest-api failed: %w", err)
	}

	if dbErr := h.Store.CompleteIngestionRun(ctx, runID, database.RunStats{
		Fetched:    totalFetched,
		Inserted:   upserted,
		DurationMs: durationMs,
	}); dbErr != nil {
		h.Logger.Error("failed to complete ingestion run", "error", dbErr)
	}

	h.Logger.Info("ingest-api complete",
		"run_id", runID,
		"total_fetched", totalFetched,
		"upserted", upserted,
		"duration_ms", durationMs,
	)

	return &Output{
		Status:       "ok",
		RunID:        runID.String(),
		JobType:      jobType,
		TotalFetched: totalFetched,
		Upserted:     upserted,
	}, nil
}

// dateWindows returns yearly posted date windows from 2020 through the current year.
func dateWindows() [][2]string {
	now := time.Now()
	currentYear := now.Year()

	var windows [][2]string
	for year := 2020; year <= currentYear; year++ {
		from := fmt.Sprintf("01/01/%d", year)
		var to string
		if year == currentYear {
			to = now.Format("01/02/2006")
		} else {
			to = fmt.Sprintf("12/31/%d", year)
		}
		windows = append(windows, [2]string{from, to})
	}
	return windows
}

func (h *Handler) runPipeline(ctx context.Context, runID uuid.UUID, snapshotDate time.Time) (totalFetched, totalUpserted int, err error) {
	windows := dateWindows()

	for _, w := range windows {
		postedFrom, postedTo := w[0], w[1]
		h.Logger.Info("fetching window", "from", postedFrom, "to", postedTo)

		opps, rawItems, _, fetchErr := h.API.FetchAll(ctx, postedFrom, postedTo)
		if fetchErr != nil {
			return totalFetched, totalUpserted, fmt.Errorf("fetch window %s-%s: %w", postedFrom, postedTo, fetchErr)
		}

		h.Logger.Info("window fetched", "from", postedFrom, "to", postedTo, "count", len(opps))
		totalFetched += len(opps)

		// Build snap rows
		snapRows := make([]database.SnapAPIRow, 0, len(rawItems))
		for i, raw := range rawItems {
			snapRows = append(snapRows, database.SnapAPIRow{
				NoticeID:    opps[i].NoticeID,
				RawData:     raw,
				ContentHash: computeHash(raw),
			})
		}

		// Bulk insert snapshots (reconcile Lambda handles upsert to opportunities)
		if len(snapRows) > 0 {
			inserted, snapErr := h.Store.BulkInsertSnapAPI(ctx, runID, snapshotDate, snapRows)
			if snapErr != nil {
				return totalFetched, totalUpserted, fmt.Errorf("bulk insert snap_api for %s-%s: %w", postedFrom, postedTo, snapErr)
			}
			h.Logger.Info("snap_api rows inserted", "count", inserted)
		}
	}

	return totalFetched, totalUpserted, nil
}

func computeHash(data json.RawMessage) string {
	hash := sha256.Sum256(data)
	return hex.EncodeToString(hash[:])
}

func main() {
	client := samgov.NewAPIClient(apiKey, logger, db)
	h := &Handler{Store: db, Logger: logger, API: client}
	lambda.Start(h.Handle)
}
