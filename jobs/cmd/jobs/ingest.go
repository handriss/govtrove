package main

import (
	"context"
	"fmt"
	"log/slog"

	awsconfig "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/sns"
	"github.com/handriss/govtrove/jobs/internal/config"
	"github.com/handriss/govtrove/jobs/internal/database"
	"github.com/handriss/govtrove/jobs/internal/ingestion"
)

func RunIngest(ctx context.Context, cfg *config.Config, db *database.DB, logger *slog.Logger) error {
	if cfg.S3ActiveCSVKey == "" && cfg.SAMAPIKey == "" {
		return fmt.Errorf("SAM_API_KEY is required when not reading from S3")
	}

	awsCfg, err := awsconfig.LoadDefaultConfig(ctx, awsconfig.WithRegion(cfg.AWSRegion))
	if err != nil {
		return fmt.Errorf("load AWS config: %w", err)
	}

	var s3Client *s3.Client
	if cfg.S3ActiveCSVKey != "" {
		s3Client = s3.NewFromConfig(awsCfg)
		logger.Info("ingesting from S3", "bucket", cfg.S3Bucket, "key", cfg.S3ActiveCSVKey)
	}

	var snsClient *sns.Client
	if cfg.SNSTopicARN != "" {
		snsClient = sns.NewFromConfig(awsCfg)
	}

	svc := ingestion.New(cfg, db, s3Client, snsClient, logger)
	return svc.Run(ctx)
}
