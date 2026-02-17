package main

import (
	"context"
	"encoding/json"

	"github.com/aws/aws-lambda-go/lambda"
)

type Input struct {
	PipelineRunID string `json:"pipeline_run_id"`
	File          struct {
		Type   string `json:"type"`
		S3Key  string `json:"s3_key"`
		Source string `json:"source"`
	} `json:"file"`
}

type Output struct {
	Status string `json:"status"`
}

func handler(ctx context.Context, event json.RawMessage) (*Output, error) {
	var input Input
	if err := json.Unmarshal(event, &input); err != nil {
		return nil, err
	}

	// TODO: wire in business logic for archived CSV ingestion
	return &Output{Status: "ok"}, nil
}

func main() { lambda.Start(handler) }
