package main

import (
	"context"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/sns"
	appconfig "github.com/opscout/ingestion/internal/config"
	"github.com/opscout/ingestion/internal/database"
	"github.com/opscout/ingestion/internal/ingestion"
)

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	}))
	slog.SetDefault(logger)

	if err := run(logger); err != nil {
		logger.Error("ingestion failed", "error", err)
		os.Exit(1)
	}

	logger.Info("ingestion service exiting")
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

	cfg, err := appconfig.Load()
	if err != nil {
		return err
	}
	logger.Info("configuration loaded")

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

	var snsClient *sns.Client
	if cfg.SNSTopicARN != "" {
		awsCfg, err := config.LoadDefaultConfig(ctx, config.WithRegion(cfg.AWSRegion))
		if err != nil {
			logger.Warn("failed to load AWS config, notifications disabled", "error", err)
		} else {
			snsClient = sns.NewFromConfig(awsCfg)
			logger.Info("SNS client configured", "topic_arn", cfg.SNSTopicARN)
		}
	} else {
		logger.Info("SNS topic not configured, notifications disabled")
	}

	svc := ingestion.New(cfg, db, snsClient, logger)
	return svc.Run(ctx)
}
