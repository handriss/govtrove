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
	appconfig "github.com/handriss/govtrove/ingestion/internal/config"
	"github.com/handriss/govtrove/ingestion/internal/csvarchive"
	"github.com/handriss/govtrove/ingestion/internal/database"
)

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	}))
	slog.SetDefault(logger)

	if err := run(logger); err != nil {
		logger.Error("csvarchive failed", "error", err)
		os.Exit(1)
	}

	logger.Info("csvarchive exiting")
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

	cfg, err := appconfig.LoadCSVArchive()
	if err != nil {
		return err
	}
	logger.Info("configuration loaded",
		"s3_bucket", cfg.S3Bucket,
		"s3_archive_enabled", cfg.S3ArchiveEnabled,
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
		sendNotification(ctx, snsClient, cfg.SNSTopicARN, logger, fmt.Sprintf("CSV Archive FAILED: %s", err))
		return err
	}

	msg := fmt.Sprintf("CSV Archive — %s\nDuration: %s", result.Outcome, result.Duration.Round(time.Second))
	if result.FileSize > 0 {
		msg += fmt.Sprintf("\nFile size: %.1f MB", float64(result.FileSize)/1024/1024)
	}
	sendNotification(ctx, snsClient, cfg.SNSTopicARN, logger, msg)

	return nil
}

func sendNotification(ctx context.Context, snsClient *sns.Client, topicARN string, logger *slog.Logger, message string) {
	if snsClient == nil || topicARN == "" {
		return
	}
	_, err := snsClient.Publish(ctx, &sns.PublishInput{
		TopicArn: aws.String(topicARN),
		Subject:  aws.String("GovTrove CSV Archive"),
		Message:  aws.String(message),
	})
	if err != nil {
		logger.Warn("failed to send notification", "error", err)
	}
}
