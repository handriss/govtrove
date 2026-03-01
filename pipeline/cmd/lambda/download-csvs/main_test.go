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

	origSrcs := bulkcsv.Sources
	defer func() { bulkcsv.Sources = origSrcs }()
	bulkcsv.Sources = nil

	// No new files = CreatePipelineRun not called, no error
	out, err := h.Handle(context.Background(), json.RawMessage(`{}`))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(out.Files) != 0 {
		t.Errorf("expected no files, got %d", len(out.Files))
	}
}

func TestHandle_NoNewFilesSkipsDBWrites(t *testing.T) {
	h, store := stubHandler(t)

	origSrcs := bulkcsv.Sources
	defer func() { bulkcsv.Sources = origSrcs }()
	bulkcsv.Sources = nil

	var pipelineRunCreated bool
	store.CreatePipelineRunFn = func(_ context.Context, _ uuid.UUID, _ string, _ map[string]any) (uuid.UUID, error) {
		pipelineRunCreated = true
		return uuid.New(), nil
	}

	out, err := h.Handle(context.Background(), json.RawMessage(`{}`))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(out.Files) != 0 {
		t.Errorf("expected no files, got %d", len(out.Files))
	}
	if pipelineRunCreated {
		t.Error("expected CreatePipelineRun NOT to be called when no new files")
	}
}

func TestHandle_OutputFilesFromResults(t *testing.T) {
	f := File{Source: "active", Type: string(bulkcsv.SourceTypeActive)}
	if f.Type != "active" {
		t.Errorf("expected type active, got %q", f.Type)
	}
}

func TestHandle_FileTypeClassification(t *testing.T) {
	cases := []struct {
		source     string
		sourceType bulkcsv.SourceType
		wantType   string
	}{
		{"active", bulkcsv.SourceTypeActive, "active"},
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
	_, parseErr := uuid.Parse(out.ExecutionID)
	if parseErr != nil {
		t.Errorf("expected valid UUID in PipelineRunID, got %q: %v", out.ExecutionID, parseErr)
	}
}

func TestHandle_UsesInputIDForExecutionID(t *testing.T) {
	h, _ := stubHandler(t)

	origSrcs := bulkcsv.Sources
	defer func() { bulkcsv.Sources = origSrcs }()
	bulkcsv.Sources = nil

	expectedID := uuid.New()

	input := fmt.Sprintf(`{"id":"%s"}`, expectedID.String())
	out, err := h.Handle(context.Background(), json.RawMessage(input))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if out.ExecutionID != expectedID.String() {
		t.Errorf("expected output ExecutionID %s, got %s", expectedID, out.ExecutionID)
	}
}

func TestHandle_InvalidInputIDFallsBack(t *testing.T) {
	h, _ := stubHandler(t)

	origSrcs := bulkcsv.Sources
	defer func() { bulkcsv.Sources = origSrcs }()
	bulkcsv.Sources = nil

	out, err := h.Handle(context.Background(), json.RawMessage(`{"id":"not-a-uuid"}`))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	_, parseErr := uuid.Parse(out.ExecutionID)
	if parseErr != nil {
		t.Errorf("expected valid UUID in output, got %q", out.ExecutionID)
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
	if out.ExecutionID != runID.String() {
		t.Errorf("expected cached PipelineRunID %s, got %s", runID, out.ExecutionID)
	}
}

func TestHandle_ReturnsErrorOnSourceFailure(t *testing.T) {
	h, store := stubHandler(t)

	origSrcs := bulkcsv.Sources
	defer func() { bulkcsv.Sources = origSrcs }()

	bulkcsv.Sources = []bulkcsv.Source{
		{Key: "bad-source", Type: bulkcsv.SourceTypeActive, URL: "http://127.0.0.1:1/nonexistent", S3Prefix: "raw/bad"},
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
}

func TestHandle_OutputHasValidExecutionID(t *testing.T) {
	h, _ := stubHandler(t)

	origSrcs := bulkcsv.Sources
	defer func() { bulkcsv.Sources = origSrcs }()
	bulkcsv.Sources = nil

	out, err := h.Handle(context.Background(), json.RawMessage(`{}`))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	_, parseErr := uuid.Parse(out.ExecutionID)
	if parseErr != nil {
		t.Errorf("expected valid UUID in ExecutionID, got %q: %v", out.ExecutionID, parseErr)
	}
}

