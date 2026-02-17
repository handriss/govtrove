package pipeline

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/sns"
	snstypes "github.com/aws/aws-sdk-go-v2/service/sns/types"
	"github.com/google/uuid"
)

type CSVDownloadedMessage struct {
	PipelineRunID uuid.UUID `json:"pipeline_run_id"`
	Type          string    `json:"type"` // "active" or "archived"
	S3Key         string    `json:"s3_key"`
	DownloadedAt  time.Time `json:"downloaded_at"`
}

type IngestionCompletedMessage struct {
	PipelineRunID uuid.UUID `json:"pipeline_run_id"`
	Source        string    `json:"source"` // "active" or "archived"
	SnapshotDate  string    `json:"snapshot_date"`
	RunID         uuid.UUID `json:"run_id"`
}

// ReadCSVDownloadedEnv reads csv-downloaded fields from individual env vars
// set by EventBridge Pipes via SNS message attributes.
func ReadCSVDownloadedEnv() (*CSVDownloadedMessage, error) {
	s3Key := os.Getenv("PIPELINE_S3_KEY")
	if s3Key == "" {
		return nil, fmt.Errorf("PIPELINE_S3_KEY not set")
	}
	runIDStr := os.Getenv("PIPELINE_RUN_ID")
	if runIDStr == "" {
		return nil, fmt.Errorf("PIPELINE_RUN_ID not set")
	}
	runID, err := uuid.Parse(runIDStr)
	if err != nil {
		return nil, fmt.Errorf("parse PIPELINE_RUN_ID: %w", err)
	}
	return &CSVDownloadedMessage{
		PipelineRunID: runID,
		Type:          os.Getenv("PIPELINE_TYPE"),
		S3Key:         s3Key,
	}, nil
}

// ReadIngestionCompletedEnv reads ingestion-completed fields from individual env vars
// set by EventBridge Pipes via SNS message attributes.
func ReadIngestionCompletedEnv() (*IngestionCompletedMessage, error) {
	runIDStr := os.Getenv("PIPELINE_RUN_ID")
	if runIDStr == "" {
		return nil, fmt.Errorf("PIPELINE_RUN_ID not set")
	}
	runID, err := uuid.Parse(runIDStr)
	if err != nil {
		return nil, fmt.Errorf("parse PIPELINE_RUN_ID: %w", err)
	}
	parentRunIDStr := os.Getenv("PIPELINE_PARENT_RUN_ID")
	parentRunID, _ := uuid.Parse(parentRunIDStr) // may be empty

	return &IngestionCompletedMessage{
		PipelineRunID: parentRunID,
		Source:        os.Getenv("PIPELINE_SOURCE"),
		SnapshotDate:  os.Getenv("PIPELINE_SNAPSHOT_DATE"),
		RunID:         runID,
	}, nil
}

func PublishCSVDownloaded(ctx context.Context, client *sns.Client, topicARN string, msg CSVDownloadedMessage) error {
	body, err := json.Marshal(msg)
	if err != nil {
		return fmt.Errorf("marshal csv-downloaded message: %w", err)
	}

	_, err = client.Publish(ctx, &sns.PublishInput{
		TopicArn: aws.String(topicARN),
		Message:  aws.String(string(body)),
		MessageAttributes: map[string]snstypes.MessageAttributeValue{
			"type": {
				DataType:    aws.String("String"),
				StringValue: aws.String(msg.Type),
			},
			"s3_key": {
				DataType:    aws.String("String"),
				StringValue: aws.String(msg.S3Key),
			},
			"pipeline_run_id": {
				DataType:    aws.String("String"),
				StringValue: aws.String(msg.PipelineRunID.String()),
			},
		},
	})
	if err != nil {
		return fmt.Errorf("publish csv-downloaded: %w", err)
	}
	return nil
}

func PublishIngestionCompleted(ctx context.Context, client *sns.Client, topicARN string, msg IngestionCompletedMessage) error {
	body, err := json.Marshal(msg)
	if err != nil {
		return fmt.Errorf("marshal ingestion-completed message: %w", err)
	}

	_, err = client.Publish(ctx, &sns.PublishInput{
		TopicArn: aws.String(topicARN),
		Message:  aws.String(string(body)),
		MessageAttributes: map[string]snstypes.MessageAttributeValue{
			"source": {
				DataType:    aws.String("String"),
				StringValue: aws.String(msg.Source),
			},
			"snapshot_date": {
				DataType:    aws.String("String"),
				StringValue: aws.String(msg.SnapshotDate),
			},
			"pipeline_run_id": {
				DataType:    aws.String("String"),
				StringValue: aws.String(msg.RunID.String()),
			},
			"parent_run_id": {
				DataType:    aws.String("String"),
				StringValue: aws.String(msg.PipelineRunID.String()),
			},
		},
	})
	if err != nil {
		return fmt.Errorf("publish ingestion-completed: %w", err)
	}
	return nil
}
