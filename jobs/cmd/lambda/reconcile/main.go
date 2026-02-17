package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"os"

	"github.com/aws/aws-lambda-go/lambda"
)

var logger *slog.Logger

func init() {
	logger = slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo}))
	slog.SetDefault(logger)
}

type Input struct {
	PipelineRunID    string            `json:"pipeline_run_id"`
	Files            []json.RawMessage `json:"files"`
	IngestionResults []struct {
		Status string `json:"status"`
	} `json:"ingestion_results"`
}

type Output struct {
	Status string `json:"status"`
}

func handler(ctx context.Context, event json.RawMessage) (*Output, error) {
	var input Input
	if err := json.Unmarshal(event, &input); err != nil {
		return nil, fmt.Errorf("unmarshal input: %w", err)
	}

	logger.Info("reconcile started",
		"pipeline_run_id", input.PipelineRunID,
		"ingestion_count", len(input.IngestionResults),
	)

	for i, r := range input.IngestionResults {
		if r.Status != "ok" {
			return nil, fmt.Errorf("ingestion %d failed with status %q", i, r.Status)
		}
	}

	logger.Info("reconcile complete — all ingestions succeeded")
	return &Output{Status: "ok"}, nil
}

func main() { lambda.Start(handler) }
