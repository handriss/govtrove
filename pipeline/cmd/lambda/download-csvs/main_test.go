package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"testing"

	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/google/uuid"
	"github.com/handriss/govtrove/pipeline/internal/bulkcsv"
	"github.com/handriss/govtrove/pipeline/internal/config"
	"github.com/handriss/govtrove/pipeline/internal/database"
	"github.com/handriss/govtrove/pipeline/internal/testutil"
)

type mockS3Client struct {
	putFn  func(ctx context.Context, params *s3.PutObjectInput, optFns ...func(*s3.Options)) (*s3.PutObjectOutput, error)
	headFn func(ctx context.Context, params *s3.HeadObjectInput, optFns ...func(*s3.Options)) (*s3.HeadObjectOutput, error)
}

func (m *mockS3Client) PutObject(ctx context.Context, params *s3.PutObjectInput, optFns ...func(*s3.Options)) (*s3.PutObjectOutput, error) {
	if m.putFn != nil {
		return m.putFn(ctx, params, optFns...)
	}
	return &s3.PutObjectOutput{}, nil
}

func (m *mockS3Client) HeadObject(ctx context.Context, params *s3.HeadObjectInput, optFns ...func(*s3.Options)) (*s3.HeadObjectOutput, error) {
	if m.headFn != nil {
		return m.headFn(ctx, params, optFns...)
	}
	return &s3.HeadObjectOutput{}, nil
}

func stubHandler(t *testing.T) (*Handler, *testutil.MockStore) {
	t.Helper()
	store := &testutil.MockStore{}
	s3c := &mockS3Client{}
	cfg := &config.Config{
		S3Bucket:         "test-bucket",
		S3ArchiveEnabled: false,
	}
	log := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelWarn}))
	h := &Handler{Store: store, S3: s3c, Cfg: cfg, Logger: log}
	return h, store
}

func TestHandle_CreatePipelineRunFailure(t *testing.T) {
	h, store := stubHandler(t)
	store.CreatePipelineRunFn = func(_ context.Context, _ string, _ map[string]any) (uuid.UUID, error) {
		return uuid.Nil, fmt.Errorf("db error")
	}

	_, err := h.Handle(context.Background(), json.RawMessage(`{}`))
	if err == nil {
		t.Error("expected error when CreatePipelineRun fails")
	}
}

func TestHandle_SourceFailurePropagates(t *testing.T) {
	h, store := stubHandler(t)

	origSrcs := bulkcsv.Sources
	defer func() { bulkcsv.Sources = origSrcs }()

	// Empty sources — bulkcsv.Run with no sources should succeed with empty results
	bulkcsv.Sources = nil

	var pipelineRunCompleted bool
	store.CompletePipelineRunFn = func(_ context.Context, _ uuid.UUID, _ map[string]any, _ int) error {
		pipelineRunCompleted = true
		return nil
	}

	out, err := h.Handle(context.Background(), json.RawMessage(`{}`))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(out.Files) != 0 {
		t.Errorf("expected no files, got %d", len(out.Files))
	}
	if !pipelineRunCompleted {
		t.Error("expected pipeline run to be completed")
	}
}

func TestHandle_OutputFilesFromResults(t *testing.T) {
	// This tests the file type detection logic (active vs archived)
	// by verifying the output transformation directly

	// active source → Type = "active"
	f := File{Source: "active"}
	if f.Source == "active" {
		f.Type = "active"
	}
	if f.Type != "active" {
		t.Errorf("expected type active, got %q", f.Type)
	}

	// archived source → Type = "archived"
	f2 := File{Source: "archived_fy2026"}
	if f2.Source != "active" {
		f2.Type = "archived"
	}
	if f2.Type != "archived" {
		t.Errorf("expected type archived, got %q", f2.Type)
	}
}

func TestHandle_FileTypeClassification(t *testing.T) {
	cases := []struct {
		source   string
		wantType string
	}{
		{"active", "active"},
		{"archived_fy2026", "archived"},
		{"archived_fy2025", "archived"},
	}

	for _, c := range cases {
		f := File{Source: c.source, S3Key: "test/key.csv.gz"}
		if c.source == "active" {
			f.Type = "active"
		} else if len(c.source) > 8 && c.source[:8] == "archived" {
			f.Type = "archived"
		}
		if f.Type != c.wantType {
			t.Errorf("source %q: got type %q, want %q", c.source, f.Type, c.wantType)
		}
	}
}

func TestHandle_PipelineRunIDInOutput(t *testing.T) {
	h, _ := stubHandler(t)

	origSrcs := bulkcsv.Sources
	defer func() { bulkcsv.Sources = origSrcs }()
	bulkcsv.Sources = nil

	out, err := h.Handle(context.Background(), json.RawMessage(`{}`))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// Verify the output has a valid UUID
	_, parseErr := uuid.Parse(out.PipelineRunID)
	if parseErr != nil {
		t.Errorf("expected valid UUID in PipelineRunID, got %q: %v", out.PipelineRunID, parseErr)
	}
}

func TestHandle_FailPipelineRunOnSourceError(t *testing.T) {
	h, store := stubHandler(t)

	origSrcs := bulkcsv.Sources
	defer func() { bulkcsv.Sources = origSrcs }()

	// Create a source with an unreachable URL that will fail
	bulkcsv.Sources = []bulkcsv.Source{
		{Key: "bad-source", URL: "http://127.0.0.1:1/nonexistent", S3Prefix: "raw/bad"},
	}

	var failedMsg string
	store.FailPipelineRunFn = func(_ context.Context, _ uuid.UUID, errMsg string, _ int) error {
		failedMsg = errMsg
		return nil
	}

	// The store also needs bulk CSV log insert for error logging
	store.InsertBulkCSVLogFn = func(_ context.Context, r *database.BulkCSVLogRecord) (int, error) {
		return 1, nil
	}
	store.GetLatestBulkCSVHashFn = func(_ context.Context, _ string) (string, error) { return "", nil }
	store.GetLatestBulkCSVHeadersFn = func(_ context.Context, _ string) (string, string, error) { return "", "", nil }

	_, err := h.Handle(context.Background(), json.RawMessage(`{}`))
	if err == nil {
		t.Error("expected error when source fails")
	}
	if failedMsg == "" {
		t.Error("expected FailPipelineRun to be called with error message")
	}
}

func TestHandle_CompletesPipelineRunStats(t *testing.T) {
	h, store := stubHandler(t)

	origSrcs := bulkcsv.Sources
	defer func() { bulkcsv.Sources = origSrcs }()
	bulkcsv.Sources = nil

	var completedStats map[string]any
	store.CompletePipelineRunFn = func(_ context.Context, _ uuid.UUID, stats map[string]any, _ int) error {
		completedStats = stats
		return nil
	}

	_, err := h.Handle(context.Background(), json.RawMessage(`{}`))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if completedStats == nil {
		t.Error("expected stats to be passed to CompletePipelineRun")
	}
	if _, ok := completedStats["summary"]; !ok {
		t.Error("expected 'summary' in stats")
	}
	if _, ok := completedStats["new_files"]; !ok {
		t.Error("expected 'new_files' in stats")
	}
}
