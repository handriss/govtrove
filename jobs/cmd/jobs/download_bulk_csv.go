package main

import (
	"context"
	"fmt"
	"log/slog"
	"strings"

	"github.com/aws/aws-sdk-go-v2/aws"
	awsconfig "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/sns"
	"github.com/handriss/govtrove/jobs/internal/bulkcsv"
	"github.com/handriss/govtrove/jobs/internal/config"
	"github.com/handriss/govtrove/jobs/internal/database"
)

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

func RunDownloadBulkCSV(ctx context.Context, cfg *config.Config, db *database.DB, logger *slog.Logger) error {
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

	results, err := bulkcsv.Run(ctx, cfg, db, s3Client, logger)

	if err != nil {
		sendNotification(ctx, snsClient, cfg.SNSTopicARN, logger,
			"GovTrove Bulk CSV",
			fmt.Sprintf("Bulk CSV FAILED: %s", err))
		return err
	}

	// Notify if any source produced a new file or error
	var hasNewFile, hasError bool
	for _, r := range results {
		if r.Outcome == "new_file" {
			hasNewFile = true
		}
		if r.Outcome == "error" {
			hasError = true
		}
	}

	if hasNewFile || hasError {
		var subject string
		if hasError {
			subject = "GovTrove Bulk CSV (errors)"
		} else {
			subject = "GovTrove Bulk CSV"
		}
		sendNotification(ctx, snsClient, cfg.SNSTopicARN, logger,
			subject,
			fmt.Sprintf("Bulk CSV results:\n%s", bulkcsv.FormatSummary(results)))
	}

	// Return error if any source failed
	if hasError {
		var failed []string
		for _, r := range results {
			if r.Outcome == "error" {
				failed = append(failed, r.Source)
			}
		}
		return fmt.Errorf("sources failed: %s", strings.Join(failed, ", "))
	}

	return nil
}
