package main

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	"github.com/handriss/govtrove/jobs/internal/config"
	"github.com/handriss/govtrove/jobs/internal/database"
)

func main() {
	if len(os.Args) < 2 {
		printUsage()
		os.Exit(1)
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		sig := <-sigCh
		slog.Info("received shutdown signal", "signal", sig)
		cancel()
	}()

	cfg, err := config.Load()
	if err != nil {
		slog.Error("failed to load config", "error", err)
		os.Exit(1)
	}

	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: config.ParseLogLevel(cfg.LogLevel),
	}))
	slog.SetDefault(logger)

	db, err := database.New(ctx, cfg.DatabaseURL)
	if err != nil {
		logger.Error("failed to connect to database", "error", err)
		os.Exit(1)
	}
	defer db.Close()
	logger.Info("connected to database")

	command := os.Args[1]
	var runErr error

	switch command {
	case "ingest":
		runErr = RunIngest(ctx, cfg, db, logger)
	case "download-bulk-csv-active":
		runErr = RunDownloadBulkCSVActive(ctx, cfg, db, logger)
	case "download-bulk-csv-archived":
		runErr = RunDownloadBulkCSVArchived(ctx, cfg, db, logger)
	default:
		fmt.Fprintf(os.Stderr, "unknown command: %s\n", command)
		printUsage()
		os.Exit(1)
	}

	if runErr != nil {
		logger.Error("job failed", "command", command, "error", runErr)
		os.Exit(1)
	}

	logger.Info("job completed", "command", command)
}

func printUsage() {
	fmt.Fprintln(os.Stderr, "Usage: jobs <command>")
	fmt.Fprintln(os.Stderr, "")
	fmt.Fprintln(os.Stderr, "Commands:")
	fmt.Fprintln(os.Stderr, "  ingest                       Run snapshot-based CSV ingestion")
	fmt.Fprintln(os.Stderr, "  download-bulk-csv-active     Download active opportunities CSV to S3")
	fmt.Fprintln(os.Stderr, "  download-bulk-csv-archived   Download archived opportunities CSVs to S3")
}
