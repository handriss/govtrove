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
	"github.com/handriss/govtrove/pipeline/internal/config"
	"github.com/handriss/govtrove/pipeline/internal/database"
	"github.com/handriss/govtrove/pipeline/internal/reconcile"
	"github.com/handriss/govtrove/pipeline/internal/samgov"
)

const (
	envDatabaseURLSecretARN = "DATABASE_URL_SECRET_ARN"
	envAWSRegion            = "AWS_REGION_NAME"
	envSentryDSN            = "SENTRY_DSN"
)

var (
	db     *database.DB
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
	secretARN := os.Getenv(envDatabaseURLSecretARN)
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
	ExecutionID      string            `json:"execution_id"`
	Files            []json.RawMessage `json:"files"`
	IngestionResults []IngestionResult `json:"ingestion_results"`
	APIResult        *IngestionResult  `json:"api_result"`
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
	start := time.Now()
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

	h.Logger.Info("reconcile started",
		"execution_id", input.ExecutionID,
		"ingestion_count", len(input.IngestionResults),
		"has_api_result", input.APIResult != nil,
	)

	// Pipeline step tracking
	var executionID *uuid.UUID
	if input.ExecutionID != "" {
		parsed, err := uuid.Parse(input.ExecutionID)
		if err == nil {
			executionID = &parsed
		}
	}

	var stepID uuid.UUID
	if executionID != nil {
		sid, err := h.Store.CreatePipelineStep(ctx, *executionID, "reconcile")
		if err != nil {
			h.Logger.Warn("failed to create pipeline step", "error", err)
		} else {
			stepID = sid
		}
	}

	defer func() {
		if retErr != nil && stepID != uuid.Nil {
			_ = h.Store.FailPipelineStep(ctx, stepID, retErr.Error(), int(time.Since(start).Milliseconds()))
		}
	}()

	// 1. Parse CSV ingestion results
	var activeRunID uuid.UUID
	for i, r := range input.IngestionResults {
		if r.Status != "ok" {
			return nil, fmt.Errorf("ingestion %d failed with status %q", i, r.Status)
		}
		rid, err := uuid.Parse(r.RunID)
		if err != nil {
			return nil, fmt.Errorf("parse run_id for ingestion %d: %w", i, err)
		}
		if r.JobType == "snapshot-csv" {
			activeRunID = rid
		}
	}

	// 2. Extract API run ID
	var apiRunID uuid.UUID
	if input.APIResult != nil {
		if input.APIResult.Status != "ok" {
			return nil, fmt.Errorf("api ingestion failed with status %q", input.APIResult.Status)
		}
		var err error
		apiRunID, err = uuid.Parse(input.APIResult.RunID)
		if err != nil {
			return nil, fmt.Errorf("parse api run_id: %w", err)
		}
	}

	snapshotDate := time.Now().UTC()

	// 3. Load CSV opps
	csvOpps := make(map[string]reconcile.Opportunity)
	var allDQEntries []database.DataQualityEntry

	if activeRunID != uuid.Nil {
		opps, dq, err := h.loadCSVOpps(ctx, activeRunID, snapshotDate)
		if err != nil {
			return nil, fmt.Errorf("load active: %w", err)
		}
		allDQEntries = append(allDQEntries, dq...)
		for _, opp := range opps {
			csvOpps[opp.NoticeID] = opp
		}
		h.Logger.Info("active CSV loaded", "run_id", activeRunID, "count", len(opps))
	}

	// 4. Load API opps
	apiOpps := make(map[string]reconcile.Opportunity)
	if apiRunID != uuid.Nil {
		rawRows, err := h.Store.GetSnapAPIRawData(ctx, apiRunID)
		if err != nil {
			return nil, fmt.Errorf("load snap_api raw_data: %w", err)
		}
		for _, row := range rawRows {
			var d samgov.OpportunityData
			if err := json.Unmarshal(row.RawData, &d); err != nil {
				h.Logger.Warn("skipping unparseable snap_api row", "notice_id", row.NoticeID, "error", err)
				continue
			}
			opp, issues := reconcile.FromAPI(d)
			apiOpps[opp.NoticeID] = opp
			for _, iss := range issues {
				allDQEntries = append(allDQEntries, database.DataQualityEntry{
					NoticeID:     opp.NoticeID,
					SnapshotDate: snapshotDate,
					Source:       "api",
					IssueType:    iss.IssueType,
					FieldName:    iss.FieldName,
					FieldValue:   iss.FieldValue,
				})
			}
		}
		h.Logger.Info("API opps loaded", "run_id", apiRunID, "count", len(apiOpps))
	}

	// 5. Insert all DQ entries
	if len(allDQEntries) > 0 {
		runID := activeRunID
		if runID == uuid.Nil && apiRunID != uuid.Nil {
			runID = apiRunID
		}
		h.Logger.Warn("data quality issues found", "count", len(allDQEntries))
		h.Store.InsertDataQualityIssues(ctx, runID, allDQEntries)
	}

	// 6. Reconcile: merge CSV + API by notice_id
	noticeIDs := make(map[string]struct{})
	for id := range csvOpps {
		noticeIDs[id] = struct{}{}
	}
	for id := range apiOpps {
		noticeIDs[id] = struct{}{}
	}

	reconciled := make([]reconcile.Opportunity, 0, len(noticeIDs))
	var reconcileDQ []database.ReconcileDQEntry
	for id := range noticeIDs {
		var csvPtr, apiPtr *reconcile.Opportunity
		if opp, ok := csvOpps[id]; ok {
			csvPtr = &opp
		}
		if opp, ok := apiOpps[id]; ok {
			apiPtr = &opp
		}
		merged, mismatches := reconcile.ReconcileRecord(csvPtr, apiPtr)
		reconciled = append(reconciled, merged)
		for _, mm := range mismatches {
			reconcileDQ = append(reconcileDQ, database.ReconcileDQEntry{
				NoticeID:     id,
				SnapshotDate: snapshotDate,
				IssueType:    mm.IssueType,
				FieldName:    mm.FieldName,
				CSVValue:     mm.CSVValue,
				APIValue:     mm.APIValue,
			})
		}
	}

	if len(reconcileDQ) > 0 {
		h.Logger.Warn("reconcile mismatches found", "count", len(reconcileDQ))
		h.Store.InsertReconcileDQIssues(ctx, activeRunID, apiRunID, reconcileDQ)
	}

	// 7. Partition reconciled into changed vs unchanged, upsert only changed
	upsertRunID := activeRunID
	if upsertRunID == uuid.Nil {
		upsertRunID = apiRunID
	}
	var totalUpserted, totalTouched int
	if len(reconciled) > 0 && upsertRunID != uuid.Nil {
		existingHashes, err := h.Store.GetExistingOpportunityHashes(ctx)
		if err != nil {
			h.Logger.Warn("failed to load existing hashes, falling back to full upsert", "error", err)
			existingHashes = nil
		}

		var changed []reconcile.Opportunity
		var unchanged []database.UnchangedRecord

		if existingHashes != nil {
			for _, opp := range reconciled {
				hash := opp.ContentHash()
				if prevHash, exists := existingHashes[opp.NoticeID]; exists && prevHash == hash {
					unchanged = append(unchanged, database.UnchangedRecord{
						NoticeID:    opp.NoticeID,
						Active:      opp.Active,
						DataSources: opp.DataSources,
					})
				} else {
					changed = append(changed, opp)
				}
			}
		} else {
			changed = reconciled
		}

		h.Logger.Info("change detection complete",
			"changed", len(changed),
			"unchanged", len(unchanged),
			"total", len(reconciled),
		)

		if len(unchanged) > 0 {
			touched, err := h.Store.BulkTouchUnchanged(ctx, upsertRunID, snapshotDate, unchanged)
			if err != nil {
				return nil, fmt.Errorf("bulk touch unchanged: %w", err)
			}
			totalTouched = touched
		}

		if len(changed) > 0 {
			upserted, err := h.Store.UpsertOpportunities(ctx, upsertRunID, snapshotDate, changed)
			if err != nil {
				return nil, fmt.Errorf("upsert opportunities: %w", err)
			}
			totalUpserted = upserted
		}
	}

	// 8. Disappearances
	if activeRunID != uuid.Nil {
		deactivated, err := h.Store.MarkDisappearedInactive(ctx, activeRunID)
		if err != nil {
			h.Logger.Error("failed to mark disappeared inactive", "error", err)
		} else if deactivated > 0 {
			h.Logger.Info("disappearances deactivated", "count", deactivated)
		}
	}

	// 9. Deactivate expired/stale opportunities
	expired, stale, err := h.Store.DeactivateExpiredOpportunities(ctx)
	if err != nil {
		h.Logger.Error("failed to deactivate expired opportunities", "error", err)
	} else if expired > 0 || stale > 0 {
		h.Logger.Info("expired opportunities deactivated", "expired", expired, "stale", stale)
	}

	durationMs := int(time.Since(start).Milliseconds())

	h.Logger.Info("reconcile complete",
		"upserted", totalUpserted,
		"touched", totalTouched,
		"csv_count", len(csvOpps),
		"api_count", len(apiOpps),
		"duration_ms", durationMs,
	)

	if stepID != uuid.Nil {
		stepStats := map[string]any{
			"upserted":       totalUpserted,
			"touched":        totalTouched,
			"csv_count":      len(csvOpps),
			"api_count":      len(apiOpps),
			"dq_issues":      len(allDQEntries),
			"reconcile_dq":   len(reconcileDQ),
		}
		completionCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if err := h.Store.CompletePipelineStep(completionCtx, stepID, stepStats, durationMs); err != nil {
			h.Logger.Warn("failed to complete pipeline step", "error", err)
		}
	}

	if refreshed, err := h.Store.RefreshAgencies(ctx); err != nil {
		h.Logger.Error("failed to refresh agencies", "error", err)
	} else {
		h.Logger.Info("agencies refreshed", "count", refreshed)
	}

	if err := h.Store.RefreshCodeCorrelations(ctx); err != nil {
		h.Logger.Error("failed to refresh code correlations", "error", err)
	} else {
		h.Logger.Info("code correlations refreshed")
	}

	if deleted, err := h.Store.DeleteOldSearchEvents(ctx, 90); err != nil {
		h.Logger.Error("failed to delete old search events", "error", err)
	} else if deleted > 0 {
		h.Logger.Info("old search events deleted", "count", deleted)
	}

	return &Output{Status: "ok"}, nil
}

func (h *Handler) loadCSVOpps(ctx context.Context, runID uuid.UUID, snapshotDate time.Time) ([]reconcile.Opportunity, []database.DataQualityEntry, error) {
	rows, err := h.Store.GetSnapCSVRawData(ctx, runID)
	if err != nil {
		return nil, nil, fmt.Errorf("load snap_csv raw_data: %w", err)
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

	return opps, dqEntries, nil
}

func main() {
	h := &Handler{Store: db, Logger: logger}
	lambda.Start(h.Handle)
}
