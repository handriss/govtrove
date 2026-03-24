package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"time"

	awsconfig "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/secretsmanager"
	"github.com/aws/aws-lambda-go/lambda"
	"github.com/getsentry/sentry-go"
	"github.com/google/uuid"

	"github.com/handriss/govtrove/pipeline/internal/database"
	"github.com/handriss/govtrove/pipeline/internal/seotrends"
)

var (
	db       *database.DB
	s3Client *s3.Client
	s3Bucket string
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
	s3Bucket = os.Getenv("S3_BUCKET")

	logger.Info("cold start complete")
}

type Input struct {
	ExecutionID string `json:"execution_id"`
}

type Output struct {
	Status        string `json:"status"`
	NAICSMovers   int    `json:"naics_movers"`
	AgencyMovers  int    `json:"agency_movers"`
	TitlePhrases  int    `json:"title_phrases"`
	SetAsideCodes int    `json:"set_aside_codes"`
}

type S3Putter interface {
	PutObject(ctx context.Context, params *s3.PutObjectInput, optFns ...func(*s3.Options)) (*s3.PutObjectOutput, error)
}

type Handler struct {
	Store    database.Store
	S3       S3Putter
	S3Bucket string
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

	var input Input
	if err := json.Unmarshal(event, &input); err != nil {
		h.Logger.Warn("failed to parse input", "error", err)
	}

	var stepID uuid.UUID
	if input.ExecutionID != "" {
		execID, err := uuid.Parse(input.ExecutionID)
		if err == nil && h.Store != nil {
			sid, err := h.Store.CreatePipelineStep(ctx, execID, "seo-trends")
			if err != nil {
				h.Logger.Warn("failed to create pipeline step", "error", err)
			} else {
				stepID = sid
			}
		}
	}

	defer func() {
		if retErr != nil && stepID != uuid.Nil {
			_ = h.Store.FailPipelineStep(ctx, stepID, retErr.Error(), int(time.Since(start).Milliseconds()))
		}
	}()

	naicsVolumes, err := h.Store.GetNAICSVolumeDelta(ctx)
	if err != nil {
		return nil, fmt.Errorf("get NAICS volumes: %w", err)
	}

	agencyVolumes, err := h.Store.GetAgencyVolumeDelta(ctx)
	if err != nil {
		return nil, fmt.Errorf("get agency volumes: %w", err)
	}

	titles, err := h.Store.GetRecentTitles(ctx)
	if err != nil {
		return nil, fmt.Errorf("get recent titles: %w", err)
	}

	setAsides, err := h.Store.GetSetAsideCounts(ctx)
	if err != nil {
		return nil, fmt.Errorf("get set-aside counts: %w", err)
	}

	trends := seotrends.Analyze(naicsVolumes, agencyVolumes, titles, setAsides)

	data, err := json.MarshalIndent(trends, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("marshal trends: %w", err)
	}

	today := time.Now().UTC().Format("2006-01-02")
	keys := []string{
		fmt.Sprintf("seo/trends/%s.json", today),
		"seo/trends/latest.json",
	}
	contentType := "application/json"

	for _, key := range keys {
		_, err := h.S3.PutObject(ctx, &s3.PutObjectInput{
			Bucket:      &h.S3Bucket,
			Key:         &key,
			Body:        bytes.NewReader(data),
			ContentType: &contentType,
		})
		if err != nil {
			return nil, fmt.Errorf("put S3 object %s: %w", key, err)
		}
	}

	durationMs := int(time.Since(start).Milliseconds())

	h.Logger.Info("seo trends complete",
		"naics_movers", len(trends.NAICSMovers),
		"agency_movers", len(trends.AgencyMovers),
		"title_phrases", len(trends.TitlePhrases),
		"set_aside_codes", len(trends.SetAsideCounts),
		"duration_ms", durationMs,
	)

	if stepID != uuid.Nil {
		stepStats := map[string]any{
			"naics_movers":   len(trends.NAICSMovers),
			"agency_movers":  len(trends.AgencyMovers),
			"title_phrases":  len(trends.TitlePhrases),
			"set_aside_codes": len(trends.SetAsideCounts),
		}
		completionCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if err := h.Store.CompletePipelineStep(completionCtx, stepID, stepStats, durationMs); err != nil {
			h.Logger.Warn("failed to complete pipeline step", "error", err)
		}
	}

	return &Output{
		Status:        "ok",
		NAICSMovers:   len(trends.NAICSMovers),
		AgencyMovers:  len(trends.AgencyMovers),
		TitlePhrases:  len(trends.TitlePhrases),
		SetAsideCodes: len(trends.SetAsideCounts),
	}, nil
}

func main() {
	h := &Handler{
		Store:    db,
		S3:       s3Client,
		S3Bucket: s3Bucket,
		Logger:   logger,
	}
	lambda.Start(h.Handle)
}
