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
	"github.com/getsentry/sentry-go"
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
		return
	}

	if dsn := os.Getenv("SENTRY_DSN"); dsn != "" {
		sentry.Init(sentry.ClientOptions{
			Dsn:              dsn,
			Environment:      "production",
			AttachStacktrace: true,
		})
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

// Handler holds dependencies for the reconcile Lambda.
type Handler struct {
	Store  database.Store
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

	h.Logger.Info("reconcile started",
		"pipeline_run_id", input.PipelineRunID,
		"ingestion_count", len(input.IngestionResults),
	)

	var activeRunID uuid.UUID
	var archivedRunIDs []uuid.UUID
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
			archivedRunIDs = append(archivedRunIDs, rid)
		}
	}

	start := time.Now()
	snapshotDate := time.Now().UTC()
	var totalUpserted int

	// Archived first so active CSV gets the last word on the active flag
	for _, archivedRunID := range archivedRunIDs {
		upserted, err := h.upsertFromRun(ctx, archivedRunID, snapshotDate)
		if err != nil {
			return nil, fmt.Errorf("upsert archived %s: %w", archivedRunID, err)
		}
		totalUpserted += upserted
		h.Logger.Info("archived opportunities upserted", "run_id", archivedRunID, "count", upserted)
	}

	if activeRunID != uuid.Nil {
		upserted, err := h.upsertFromRun(ctx, activeRunID, snapshotDate)
		if err != nil {
			return nil, fmt.Errorf("upsert active: %w", err)
		}
		totalUpserted += upserted
		h.Logger.Info("active opportunities upserted", "run_id", activeRunID, "count", upserted)
	}

	if activeRunID != uuid.Nil {
		for _, archivedRunID := range archivedRunIDs {
			resolved, err := h.Store.ResolveExpectedDisappearances(ctx, activeRunID, archivedRunID, snapshotDate)
			if err != nil {
				h.Logger.Error("failed to resolve expected disappearances", "error", err, "archived_run_id", archivedRunID)
			} else if resolved > 0 {
				h.Logger.Info("expected disappearances resolved (archived)", "count", resolved, "archived_run_id", archivedRunID)
			}
		}

		deactivated, err := h.Store.MarkDisappearedInactive(ctx, activeRunID)
		if err != nil {
			h.Logger.Error("failed to mark disappeared inactive", "error", err)
		} else if deactivated > 0 {
			h.Logger.Info("unexpected disappearances deactivated", "count", deactivated)
		}
	}

	h.Logger.Info("reconcile complete",
		"upserted", totalUpserted,
		"duration_ms", time.Since(start).Milliseconds(),
	)

	return &Output{Status: "ok"}, nil
}

func (h *Handler) upsertFromRun(ctx context.Context, runID uuid.UUID, snapshotDate time.Time) (int, error) {
	rows, err := h.Store.GetSnapCSVRawData(ctx, runID)
	if err != nil {
		return 0, fmt.Errorf("load snap_csv raw_data: %w", err)
	}

	opps := make([]reconcile.Opportunity, 0, len(rows))
	var dqEntries []database.DataQualityEntry
	for _, raw := range rows {
		opp, issues := reconcile.FromCSV(raw)
		opps = append(opps, opp)
		for _, iss := range issues {
			dqEntries = append(dqEntries, database.DataQualityEntry{
				NoticeID:     opp.NoticeID,
				SnapshotDate: snapshotDate,
				Source:       "csv",
				IssueType:    iss.IssueType,
				FieldName:    iss.FieldName,
				FieldValue:   iss.FieldValue,
			})
		}
	}

	if len(dqEntries) > 0 {
		h.Logger.Warn("data quality issues found", "count", len(dqEntries), "run_id", runID)
		h.Store.InsertDataQualityIssues(ctx, runID, dqEntries)
	}

	affected, err := h.Store.UpsertOpportunities(ctx, runID, snapshotDate, opps)
	if err != nil {
		return 0, fmt.Errorf("upsert opportunities: %w", err)
	}

	return affected, nil
}

func main() {
	h := &Handler{Store: db, Logger: logger}
	lambda.Start(h.Handle)
}
