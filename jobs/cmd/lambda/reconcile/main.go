package main

import (
	"context"
	"encoding/json"

	"github.com/aws/aws-lambda-go/lambda"
)

type Output struct {
	Status string `json:"status"`
}

func handler(ctx context.Context, event json.RawMessage) (*Output, error) {
	// TODO: wire in business logic from jobs/internal/reconcile
	return &Output{Status: "ok"}, nil
}

func main() { lambda.Start(handler) }
