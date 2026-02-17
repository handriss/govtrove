package main

import (
	"context"
	"fmt"
	"log/slog"
	"strings"

	"github.com/aws/aws-sdk-go-v2/aws"
	awsconfig "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/ecs"
	ecstypes "github.com/aws/aws-sdk-go-v2/service/ecs/types"
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

	// Trigger ingestion if the active source produced a new file
	for _, r := range results {
		if r.Source == "active" && r.Outcome == "new_file" && r.S3Key != "" {
			triggerIngestion(ctx, cfg, awsCfg, logger, r.S3Key)
			break
		}
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

func triggerIngestion(ctx context.Context, cfg *config.Config, awsCfg aws.Config, logger *slog.Logger, s3Key string) {
	if cfg.ECSCluster == "" || cfg.IngestionTaskDef == "" {
		logger.Info("ECS config not set, skipping ingestion trigger", "s3_key", s3Key)
		return
	}

	subnets := strings.Split(cfg.ECSSubnets, ",")
	ecsClient := ecs.NewFromConfig(awsCfg)

	input := &ecs.RunTaskInput{
		Cluster:        aws.String(cfg.ECSCluster),
		TaskDefinition: aws.String(cfg.IngestionTaskDef),
		LaunchType:     ecstypes.LaunchTypeFargate,
		Count:          aws.Int32(1),
		NetworkConfiguration: &ecstypes.NetworkConfiguration{
			AwsvpcConfiguration: &ecstypes.AwsVpcConfiguration{
				Subnets:        subnets,
				SecurityGroups: []string{cfg.ECSSecurityGroup},
				AssignPublicIp: ecstypes.AssignPublicIpEnabled,
			},
		},
		Overrides: &ecstypes.TaskOverride{
			ContainerOverrides: []ecstypes.ContainerOverride{
				{
					Name: aws.String("ingestion"),
					Environment: []ecstypes.KeyValuePair{
						{
							Name:  aws.String("S3_ACTIVE_CSV_KEY"),
							Value: aws.String(s3Key),
						},
						{
							Name:  aws.String("S3_BUCKET"),
							Value: aws.String(cfg.S3Bucket),
						},
					},
				},
			},
		},
	}

	out, err := ecsClient.RunTask(ctx, input)
	if err != nil {
		logger.Error("failed to trigger ingestion task", "error", err)
		return
	}

	if len(out.Tasks) > 0 {
		logger.Info("ingestion task triggered",
			"task_arn", aws.ToString(out.Tasks[0].TaskArn),
			"s3_key", s3Key,
		)
	}
	if len(out.Failures) > 0 {
		logger.Error("ingestion task launch had failures",
			"reason", aws.ToString(out.Failures[0].Reason),
		)
	}
}
