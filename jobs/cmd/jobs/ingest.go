package main

import (
	"context"
	"fmt"
	"log/slog"

	awsconfig "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/sns"
	"github.com/handriss/govtrove/jobs/internal/config"
	"github.com/handriss/govtrove/jobs/internal/database"
	"github.com/handriss/govtrove/jobs/internal/ingestion"
)

func RunIngest(ctx context.Context, cfg *config.Config, db *database.DB, logger *slog.Logger) error {
	if cfg.SAMAPIKey == "" {
		return fmt.Errorf("SAM_API_KEY is required for the ingest command")
	}

	var snsClient *sns.Client
	if cfg.SNSTopicARN != "" {
		awsCfg, err := awsconfig.LoadDefaultConfig(ctx, awsconfig.WithRegion(cfg.AWSRegion))
		if err != nil {
			return fmt.Errorf("load AWS config: %w", err)
		}
		snsClient = sns.NewFromConfig(awsCfg)
	}

	svc := ingestion.New(cfg, db, snsClient, logger)
	return svc.Run(ctx)
}
