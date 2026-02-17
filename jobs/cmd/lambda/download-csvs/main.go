package main

import (
	"context"
	"encoding/json"

	"github.com/aws/aws-lambda-go/lambda"
)

type File struct {
	Type   string `json:"type"`
	S3Key  string `json:"s3_key"`
	Source string `json:"source"`
}

type Output struct {
	PipelineRunID string `json:"pipeline_run_id"`
	Files         []File `json:"files"`
}

func handler(ctx context.Context, event json.RawMessage) (*Output, error) {
	// TODO: wire in business logic from jobs/internal/bulkcsv
	return &Output{
		PipelineRunID: "",
		Files:         nil,
	}, nil
}

func main() { lambda.Start(handler) }
