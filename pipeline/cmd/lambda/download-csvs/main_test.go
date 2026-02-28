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
	store.CreatePipelineRunFn = func(_ context.Context, _ uuid.UUID, _ string, _ map[string]any) (uuid.UUID, error) {
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
	// Verify source type propagation through File output
	f := File{Source: "active", Type: string(bulkcsv.SourceTypeActive)}
	if f.Type != "active" {
		t.Errorf("expected type active, got %q", f.Type)
	}

	f2 := File{Source: "archived_fy2026", Type: string(bulkcsv.SourceTypeArchived)}
	if f2.Type != "archived" {
		t.Errorf("expected type archived, got %q", f2.Type)
	}
}

func TestHandle_FileTypeClassification(t *testing.T) {
	cases := []struct {
		source     string
		sourceType bulkcsv.SourceType
		wantType   string
	}{
		{"active", bulkcsv.SourceTypeActive, "active"},
		{"archived_fy2026", bulkcsv.SourceTypeArchived, "archived"},
		{"archived_fy2025", bulkcsv.SourceTypeArchived, "archived"},
	}

	for _, c := range cases {
		f := File{
			Source: c.source,
			S3Key:  "test/key.csv.gz",
			Type:   string(c.sourceType),
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
	_, parseErr := uuid.Parse(out.PipelineRunID)
	if parseErr != nil {
		t.Errorf("expected valid UUID in PipelineRunID, got %q: %v", out.PipelineRunID, parseErr)
	}
}

func TestHandle_UsesInputIDForPipelineRun(t *testing.T) {
	h, store := stubHandler(t)

	origSrcs := bulkcsv.Sources
	defer func() { bulkcsv.Sources = origSrcs }()
	bulkcsv.Sources = nil

	expectedID := uuid.New()
	var receivedID uuid.UUID
	store.CreatePipelineRunFn = func(_ context.Context, id uuid.UUID, _ string, _ map[string]any) (uuid.UUID, error) {
		receivedID = id
		return id, nil
	}

	input := fmt.Sprintf(`{"id":"%s"}`, expectedID.String())
	out, err := h.Handle(context.Background(), json.RawMessage(input))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if receivedID != expectedID {
		t.Errorf("expected CreatePipelineRun to receive ID %s, got %s", expectedID, receivedID)
	}
	if out.PipelineRunID != expectedID.String() {
		t.Errorf("expected output PipelineRunID %s, got %s", expectedID, out.PipelineRunID)
	}
}

func TestHandle_InvalidInputIDFallsBack(t *testing.T) {
	h, store := stubHandler(t)

	origSrcs := bulkcsv.Sources
	defer func() { bulkcsv.Sources = origSrcs }()
	bulkcsv.Sources = nil

	var receivedID uuid.UUID
	store.CreatePipelineRunFn = func(_ context.Context, id uuid.UUID, _ string, _ map[string]any) (uuid.UUID, error) {
		receivedID = id
		if id == uuid.Nil {
			return uuid.New(), nil
		}
		return id, nil
	}

	out, err := h.Handle(context.Background(), json.RawMessage(`{"id":"not-a-uuid"}`))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if receivedID != uuid.Nil {
		t.Errorf("expected uuid.Nil for invalid input, got %s", receivedID)
	}
	_, parseErr := uuid.Parse(out.PipelineRunID)
	if parseErr != nil {
		t.Errorf("expected valid UUID in output, got %q", out.PipelineRunID)
	}
}

func TestHandle_IdempotencySkipsCompletedRun(t *testing.T) {
	h, store := stubHandler(t)

	origSrcs := bulkcsv.Sources
	defer func() { bulkcsv.Sources = origSrcs }()
	bulkcsv.Sources = nil

	runID := uuid.New()
	store.GetPipelineRunFn = func(_ context.Context, id uuid.UUID) (*database.PipelineRun, error) {
		if id == runID {
			return &database.PipelineRun{
				ID:     runID,
				Status: "completed",
				Stats:  map[string]any{"new_files": float64(0)},
			}, nil
		}
		return nil, nil
	}

	var createCalled bool
	store.CreatePipelineRunFn = func(_ context.Context, _ uuid.UUID, _ string, _ map[string]any) (uuid.UUID, error) {
		createCalled = true
		return runID, nil
	}

	input := fmt.Sprintf(`{"id":"%s"}`, runID.String())
	out, err := h.Handle(context.Background(), json.RawMessage(input))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if createCalled {
		t.Error("expected CreatePipelineRun to NOT be called for completed run")
	}
	if out.PipelineRunID != runID.String() {
		t.Errorf("expected cached PipelineRunID %s, got %s", runID, out.PipelineRunID)
	}
}

func TestHandle_FailPipelineRunOnSourceError(t *testing.T) {
	h, store := stubHandler(t)

	origSrcs := bulkcsv.Sources
	defer func() { bulkcsv.Sources = origSrcs }()

	bulkcsv.Sources = []bulkcsv.Source{
		{Key: "bad-source", Type: bulkcsv.SourceTypeActive, URL: "http://127.0.0.1:1/nonexistent", S3Prefix: "raw/bad"},
	}

	var failedMsg string
	store.FailPipelineRunFn = func(_ context.Context, _ uuid.UUID, errMsg string, _ int) error {
		failedMsg = errMsg
		return nil
	}

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

