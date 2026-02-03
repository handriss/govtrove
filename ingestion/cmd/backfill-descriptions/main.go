package main

import (
	"context"
	"flag"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	"github.com/opscout/ingestion/internal/database"
	"github.com/opscout/ingestion/internal/samgov"
)

func main() {
	limit := flag.Int("limit", 100, "Maximum number of descriptions to fetch")
	flag.Parse()

	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	}))

	if err := run(logger, *limit); err != nil {
		logger.Error("backfill failed", "error", err)
		os.Exit(1)
	}
}

func run(logger *slog.Logger, limit int) error {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		sig := <-sigCh
		logger.Info("received shutdown signal", "signal", sig)
		cancel()
	}()

	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		logger.Error("DATABASE_URL environment variable is required")
		os.Exit(1)
	}

	apiKey := os.Getenv("SAM_API_KEY")
	if apiKey == "" {
		logger.Error("SAM_API_KEY environment variable is required")
		os.Exit(1)
	}

	db, err := database.New(ctx, dbURL)
	if err != nil {
		return err
	}
	defer db.Close()
	logger.Info("connected to database")

	noticeIDs, err := db.GetNoticeIDsWithoutDescription(ctx, limit)
	if err != nil {
		return err
	}

	if len(noticeIDs) == 0 {
		logger.Info("no opportunities without descriptions found")
		return nil
	}

	logger.Info("found opportunities without descriptions", "count", len(noticeIDs))

	client := samgov.NewDescriptionClient(db, logger, apiKey)
	result, err := client.FetchDescriptionsForNotices(ctx, db, noticeIDs)
	if err != nil {
		logger.Error("fetch interrupted", "error", err)
	}

	logger.Info("backfill complete",
		"fetched", result.Fetched,
		"errors", result.Errors,
	)

	return nil
}
