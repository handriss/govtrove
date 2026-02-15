package main

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	awsconfig "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/sns"
	"github.com/handriss/govtrove/ingestion/internal/apiprobe"
	appconfig "github.com/handriss/govtrove/ingestion/internal/config"
	"github.com/handriss/govtrove/ingestion/internal/database"
)

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	}))
	slog.SetDefault(logger)

	if err := run(logger); err != nil {
		logger.Error("apiprobe failed", "error", err)
		os.Exit(1)
	}

	logger.Info("apiprobe exiting")
}

func run(logger *slog.Logger) error {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		sig := <-sigCh
		logger.Info("received shutdown signal", "signal", sig)
		cancel()
	}()

	cfg, err := appconfig.LoadAPIProbe()
	if err != nil {
		return err
	}
	logger.Info("configuration loaded",
		"mode", cfg.Mode,
		"s3_bucket", cfg.S3Bucket,
		"s3_archive_enabled", cfg.S3ArchiveEnabled,
		"lookback_days", cfg.LookbackDays,
	)

	var logLevel slog.Level
	switch cfg.LogLevel {
	case "debug":
		logLevel = slog.LevelDebug
	case "warn":
		logLevel = slog.LevelWarn
	case "error":
		logLevel = slog.LevelError
	default:
		logLevel = slog.LevelInfo
	}
	logger = slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: logLevel,
	}))

	db, err := database.New(ctx, cfg.DatabaseURL)
	if err != nil {
		return err
	}
	defer db.Close()
	logger.Info("connected to database")

	awsCfg, err := awsconfig.LoadDefaultConfig(ctx, awsconfig.WithRegion(cfg.AWSRegion))
	if err != nil {
		return fmt.Errorf("load AWS config: %w", err)
	}

	var snsClient *sns.Client
	if cfg.SNSTopicARN != "" {
		snsClient = sns.NewFromConfig(awsCfg)
	}

	var s3Client *s3.Client
	if cfg.S3ArchiveEnabled {
		s3Client = s3.NewFromConfig(awsCfg)
		logger.Info("S3 client configured", "bucket", cfg.S3Bucket)
	} else {
		logger.Info("S3 archive disabled")
	}

	if cfg.Mode == "archive" {
		result, err := apiprobe.RunArchive(ctx, cfg, db, s3Client, logger)
		if err != nil {
			sendNotification(ctx, snsClient, cfg.SNSTopicARN, logger, fmt.Sprintf("API Archive FAILED: %s", err))
			return err
		}

		if result.Outcome == "new_file" {
			msg := fmt.Sprintf("API Archive — new_file\nRecords: %d\nDuration: %s\nFile size: %.1f MB",
				result.RecordsFetched, result.Duration.Round(time.Second), float64(result.FileSize)/1024/1024)
			sendNotification(ctx, snsClient, cfg.SNSTopicARN, logger, msg)
		}
	} else {
		_, err := apiprobe.RunProbe(ctx, cfg, db, s3Client, logger)
		if err != nil {
			sendNotification(ctx, snsClient, cfg.SNSTopicARN, logger, fmt.Sprintf("API Probe FAILED: %s", err))
			return err
		}
	}

	return nil
}

func sendNotification(ctx context.Context, snsClient *sns.Client, topicARN string, logger *slog.Logger, message string) {
	if snsClient == nil || topicARN == "" {
		return
	}
	_, err := snsClient.Publish(ctx, &sns.PublishInput{
		TopicArn: aws.String(topicARN),
		Subject:  aws.String("GovTrove API Probe"),
		Message:  aws.String(message),
	})
	if err != nil {
		logger.Warn("failed to send notification", "error", err)
	}
}
