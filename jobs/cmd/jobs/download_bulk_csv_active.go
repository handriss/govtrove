package main

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	awsconfig "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/sns"
	"github.com/handriss/govtrove/jobs/internal/config"
	"github.com/handriss/govtrove/jobs/internal/csvarchive"
	"github.com/handriss/govtrove/jobs/internal/database"
)

func RunDownloadBulkCSVActive(ctx context.Context, cfg *config.Config, db *database.DB, logger *slog.Logger) error {
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

	result, err := csvarchive.Run(ctx, cfg, db, s3Client, logger)

	if err != nil {
		sendNotification(ctx, snsClient, cfg.SNSTopicARN, logger,
			"GovTrove CSV Archive",
			fmt.Sprintf("CSV Archive FAILED: %s", err))
		return err
	}

	if result.Outcome == "new_file" {
		sendNotification(ctx, snsClient, cfg.SNSTopicARN, logger,
			"GovTrove CSV Archive",
			fmt.Sprintf("CSV Archive — new_file\nDuration: %s\nFile size: %.1f MB",
				result.Duration.Round(time.Second), float64(result.FileSize)/1024/1024))
	}

	return nil
}

func sendNotification(ctx context.Context, snsClient *sns.Client, topicARN string, logger *slog.Logger, subject, message string) {
	if snsClient == nil || topicARN == "" {
		return
	}
	_, err := snsClient.Publish(ctx, &sns.PublishInput{
		TopicArn: aws.String(topicARN),
		Subject:  aws.String(subject),
		Message:  aws.String(message),
	})
	if err != nil {
		logger.Warn("failed to send notification", "error", err)
	}
}
