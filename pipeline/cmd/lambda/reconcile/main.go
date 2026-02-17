package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"time"

	awsconfig "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/secretsmanager"
	"github.com/aws/aws-lambda-go/lambda"
	"github.com/google/uuid"
	"github.com/handriss/govtrove/pipeline/internal/database"
	"github.com/handriss/govtrove/pipeline/internal/reconcile"
)

var (
	db     *database.DB
	logger *slog.Logger
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

	logger.Info("cold start complete")
}

type IngestionResult struct {
	Status  string `json:"status"`
	RunID   string `json:"run_id"`
	JobType string `json:"job_type"`
}

type Input struct {
	PipelineRunID    string            `json:"pipeline_run_id"`
	Files            []json.RawMessage `json:"files"`
	IngestionResults []IngestionResult `json:"ingestion_results"`
}

type Output struct {
	Status string `json:"status"`
}

func handler(ctx context.Context, event json.RawMessage) (*Output, error) {
	var input Input
	if err := json.Unmarshal(event, &input); err != nil {
		return nil, fmt.Errorf("unmarshal input: %w", err)
	}

	logger.Info("reconcile started",
		"pipeline_run_id", input.PipelineRunID,
		"ingestion_count", len(input.IngestionResults),
	)

	var activeRunID, archivedRunID uuid.UUID
	for i, r := range input.IngestionResults {
		if r.Status != "ok" {
			return nil, fmt.Errorf("ingestion %d failed with status %q", i, r.Status)
		}
		rid, err := uuid.Parse(r.RunID)
		if err != nil {
			return nil, fmt.Errorf("parse run_id for ingestion %d: %w", i, err)
		}
		switch r.JobType {
		case "snapshot-csv":
			activeRunID = rid
		case "ingest-archived":
			archivedRunID = rid
		}
	}

	start := time.Now()
	snapshotDate := time.Now().UTC()
	var totalUpserted int

	// Upsert active opportunities
	if activeRunID != uuid.Nil {
		upserted, err := upsertFromRun(ctx, activeRunID, snapshotDate)
		if err != nil {
			return nil, fmt.Errorf("upsert active: %w", err)
		}
		totalUpserted += upserted
		logger.Info("active opportunities upserted", "run_id", activeRunID, "count", upserted)
	}

	// Upsert archived opportunities
	if archivedRunID != uuid.Nil {
		upserted, err := upsertFromRun(ctx, archivedRunID, snapshotDate)
		if err != nil {
			return nil, fmt.Errorf("upsert archived: %w", err)
		}
		totalUpserted += upserted
		logger.Info("archived opportunities upserted", "run_id", archivedRunID, "count", upserted)
	}

	// Classify and handle disappearances
	if activeRunID != uuid.Nil {
		// Resolve expected disappearances: notices that moved to the archived CSV
		if archivedRunID != uuid.Nil {
			resolved, err := db.ResolveExpectedDisappearances(ctx, activeRunID, archivedRunID, snapshotDate)
			if err != nil {
				logger.Error("failed to resolve expected disappearances", "error", err)
			} else if resolved > 0 {
				logger.Info("expected disappearances resolved (archived)", "count", resolved)
			}
		}

		// Deactivate all unresolved disappearances (unexpected — not in archived CSV)
		deactivated, err := db.MarkDisappearedInactive(ctx, activeRunID)
		if err != nil {
			logger.Error("failed to mark disappeared inactive", "error", err)
		} else if deactivated > 0 {
			logger.Info("unexpected disappearances deactivated", "count", deactivated)
		}
	}

	logger.Info("reconcile complete",
		"upserted", totalUpserted,
		"duration_ms", time.Since(start).Milliseconds(),
	)

	return &Output{Status: "ok"}, nil
}

func upsertFromRun(ctx context.Context, runID uuid.UUID, snapshotDate time.Time) (int, error) {
	rows, err := db.GetSnapCSVRawData(ctx, runID)
	if err != nil {
		return 0, fmt.Errorf("load snap_csv raw_data: %w", err)
	}

	opps := make([]reconcile.Opportunity, 0, len(rows))
	for _, raw := range rows {
		opps = append(opps, reconcile.FromCSV(raw))
	}

	inserted, _, err := db.UpsertOpportunities(ctx, runID, snapshotDate, opps)
	if err != nil {
		return 0, fmt.Errorf("upsert opportunities: %w", err)
	}

	return inserted, nil
}

func main() { lambda.Start(handler) }
