package main

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	awsconfig "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/sns"
	"github.com/handriss/govtrove/jobs/internal/archivedcsv"
	"github.com/handriss/govtrove/jobs/internal/config"
	"github.com/handriss/govtrove/jobs/internal/database"
)

func RunDownloadBulkCSVArchived(ctx context.Context, cfg *config.Config, db *database.DB, logger *slog.Logger) error {
	awsCfg, err := awsconfig.LoadDefaultConfig(ctx, awsconfig.WithRegion(cfg.AWSRegion))
	if err != nil {
		return fmt.Errorf("load AWS config: %w", err)
	}

	var s3Client *s3.Client
	if cfg.S3ArchiveEnabled {
		s3Client = s3.NewFromConfig(awsCfg)
		logger.Info("S3 client configured", "bucket", cfg.S3Bucket)
	} else {
		logger.Info("S3 archive disabled")
	}

	var snsClient *sns.Client
	if cfg.SNSTopicARN != "" {
		snsClient = sns.NewFromConfig(awsCfg)
	}

	result, err := archivedcsv.Run(ctx, cfg, db, s3Client, logger)

	if err != nil {
		sendNotification(ctx, snsClient, cfg.SNSTopicARN, logger,
			"GovTrove Archived CSV",
			fmt.Sprintf("Archived CSV FAILED: %s", err))
		return err
	}

	if result.Outcome == "new_file" {
		sendNotification(ctx, snsClient, cfg.SNSTopicARN, logger,
			"GovTrove Archived CSV",
			fmt.Sprintf("Archived CSV — FY2026 new file\nDuration: %s\nSize: %.1f MB\nRows: %d",
				result.Duration.Round(time.Second),
				float64(result.FileSize)/1024/1024,
				result.RowCount))
	}

	return nil
}
