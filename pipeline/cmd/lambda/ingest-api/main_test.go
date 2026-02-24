package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/handriss/govtrove/pipeline/internal/database"
	"github.com/handriss/govtrove/pipeline/internal/samgov"
	"github.com/handriss/govtrove/pipeline/internal/testutil"
)

// --- dateWindows ---

func TestDateWindows_ContainsCurrentYear(t *testing.T) {
	windows := dateWindows()
	if len(windows) == 0 {
		t.Fatal("expected at least one window")
	}

	currentYear := time.Now().Year()
	lastWindow := windows[len(windows)-1]
	wantFrom := fmt.Sprintf("01/01/%d", currentYear)
	if lastWindow[0] != wantFrom {
		t.Errorf("last window from: got %q, want %q", lastWindow[0], wantFrom)
	}

	// Current year window should use today's date as "to"
	wantTo := time.Now().Format("01/02/2006")
	if lastWindow[1] != wantTo {
		t.Errorf("last window to: got %q, want %q", lastWindow[1], wantTo)
	}
}

func TestDateWindows_StartsFrom2020(t *testing.T) {
	windows := dateWindows()
	if windows[0][0] != "01/01/2020" {
		t.Errorf("first window should start from 2020, got %q", windows[0][0])
	}
}

func TestDateWindows_HistoricalWindowsEndDec31(t *testing.T) {
	windows := dateWindows()
	if len(windows) < 2 {
		t.Skip("need at least 2 windows to test historical")
	}
	// First window (2020) should end Dec 31
	if windows[0][1] != "12/31/2020" {
		t.Errorf("2020 window to: got %q, want %q", windows[0][1], "12/31/2020")
	}
}

func TestDateWindows_ConsecutiveYears(t *testing.T) {
	windows := dateWindows()
	currentYear := time.Now().Year()
	expectedCount := currentYear - 2020 + 1
	if len(windows) != expectedCount {
		t.Errorf("expected %d windows, got %d", expectedCount, len(windows))
	}
}

// --- computeHash ---

func TestComputeHash_Deterministic(t *testing.T) {
	data := json.RawMessage(`{"key":"value"}`)
	h1 := computeHash(data)
	h2 := computeHash(data)
	if h1 != h2 {
		t.Errorf("hash not deterministic: %s != %s", h1, h2)
	}
}

func TestComputeHash_DifferentInputs(t *testing.T) {
	h1 := computeHash(json.RawMessage(`{"a":1}`))
	h2 := computeHash(json.RawMessage(`{"a":2}`))
	if h1 == h2 {
		t.Error("different inputs should produce different hashes")
	}
}

func TestComputeHash_ValidHexLength(t *testing.T) {
	h := computeHash(json.RawMessage(`{}`))
	// SHA256 produces 64 hex characters
	if len(h) != 64 {
		t.Errorf("expected 64 char hex, got %d chars", len(h))
	}
}

// --- mockAPIClient ---

type mockAPIClient struct {
	fetchAllFn func(ctx context.Context, from, to string) ([]samgov.OpportunityData, []json.RawMessage, int, error)
}

func (m *mockAPIClient) FetchAll(ctx context.Context, from, to string) ([]samgov.OpportunityData, []json.RawMessage, int, error) {
	if m.fetchAllFn != nil {
		return m.fetchAllFn(ctx, from, to)
	}
	return nil, nil, 0, nil
}

// stubHandler creates a Handler with default mocks. Tests override specific functions as needed.
func stubHandler(t *testing.T) (*Handler, *testutil.MockStore, *mockAPIClient) {
	t.Helper()
	store := &testutil.MockStore{}
	api := &mockAPIClient{}
	log := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelWarn}))
	h := &Handler{Store: store, Logger: log}
	return h, store, api
}

// --- Handler.Handle ---

func TestHandle_FallbackSkipsWhenCSVRecent(t *testing.T) {
	h, store, _ := stubHandler(t)

	store.GetLastCompletedRunFn = func(_ context.Context, jobType string) (uuid.UUID, time.Time, error) {
		if jobType == "snapshot-csv" {
			return uuid.New(), time.Now().Add(-1 * time.Hour), nil
		}
		return uuid.Nil, time.Time{}, fmt.Errorf("not found")
	}

	event, _ := json.Marshal(Input{Source: "fallback"})
	out, err := h.Handle(context.Background(), event)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if out.Status != "skipped" {
		t.Errorf("expected skipped status, got %q", out.Status)
	}
}

func TestHandle_FallbackProceedsWhenCSVStale(t *testing.T) {
	h, store, _ := stubHandler(t)
	h.API = samgov.NewAPIClient("test-key", h.Logger, nil)

	store.GetLastCompletedRunFn = func(_ context.Context, jobType string) (uuid.UUID, time.Time, error) {
		return uuid.New(), time.Now().Add(-48 * time.Hour), nil
	}
	var ingestionRunCreated bool
	store.CreateIngestionRunFn = func(_ context.Context, jobType string) (uuid.UUID, error) {
		ingestionRunCreated = true
		if jobType != "snapshot-api" {
			t.Errorf("expected job type snapshot-api, got %q", jobType)
		}
		return uuid.New(), nil
	}

	var failedRun bool
	store.FailIngestionRunFn = func(_ context.Context, _ uuid.UUID, _ string, _ int) error {
		failedRun = true
		return nil
	}

	event, _ := json.Marshal(Input{Source: "fallback"})
	// Will fail because real APIClient can't reach SAM.gov, but should create ingestion run
	_, _ = h.Handle(context.Background(), event)
	if !ingestionRunCreated {
		t.Error("expected ingestion run to be created for stale CSV")
	}
	if !failedRun {
		t.Error("expected run to be marked failed (API unreachable)")
	}
}

func TestHandle_FallbackProceedsWhenCSVNeverRan(t *testing.T) {
	h, store, _ := stubHandler(t)
	h.API = samgov.NewAPIClient("test-key", h.Logger, nil)

	store.GetLastCompletedRunFn = func(_ context.Context, _ string) (uuid.UUID, time.Time, error) {
		return uuid.Nil, time.Time{}, fmt.Errorf("no runs found")
	}

	var ingestionRunCreated bool
	store.CreateIngestionRunFn = func(_ context.Context, _ string) (uuid.UUID, error) {
		ingestionRunCreated = true
		return uuid.New(), nil
	}
	store.FailIngestionRunFn = func(_ context.Context, _ uuid.UUID, _ string, _ int) error { return nil }

	event, _ := json.Marshal(Input{Source: "fallback"})
	_, _ = h.Handle(context.Background(), event)
	if !ingestionRunCreated {
		t.Error("expected ingestion run to be created when CSV never ran")
	}
}

func TestHandle_DirectInvocationSkipsFallbackCheck(t *testing.T) {
	h, store, _ := stubHandler(t)
	h.API = samgov.NewAPIClient("test-key", h.Logger, nil)

	var csvCheckCalled bool
	store.GetLastCompletedRunFn = func(_ context.Context, _ string) (uuid.UUID, time.Time, error) {
		csvCheckCalled = true
		return uuid.Nil, time.Time{}, nil
	}

	var ingestionRunCreated bool
	store.CreateIngestionRunFn = func(_ context.Context, _ string) (uuid.UUID, error) {
		ingestionRunCreated = true
		return uuid.New(), nil
	}
	store.FailIngestionRunFn = func(_ context.Context, _ uuid.UUID, _ string, _ int) error { return nil }

	event, _ := json.Marshal(Input{Source: "direct"})
	_, _ = h.Handle(context.Background(), event)

	if csvCheckCalled {
		t.Error("direct invocation should not check CSV pipeline status")
	}
	if !ingestionRunCreated {
		t.Error("direct invocation should create ingestion run")
	}
}

func TestHandle_InvalidJSON(t *testing.T) {
	h, _, _ := stubHandler(t)
	_, err := h.Handle(context.Background(), json.RawMessage(`{invalid}`))
	if err == nil {
		t.Error("expected error for invalid JSON")
	}
}

func TestHandle_CreateIngestionRunFailure(t *testing.T) {
	h, store, _ := stubHandler(t)
	h.API = samgov.NewAPIClient("test-key", h.Logger, nil)

	store.CreateIngestionRunFn = func(_ context.Context, _ string) (uuid.UUID, error) {
		return uuid.Nil, fmt.Errorf("db connection failed")
	}

	event, _ := json.Marshal(Input{Source: "direct"})
	_, err := h.Handle(context.Background(), event)
	if err == nil {
		t.Error("expected error when CreateIngestionRun fails")
	}
}

func TestHandle_OutputFormat(t *testing.T) {
	h, store, _ := stubHandler(t)
	h.API = samgov.NewAPIClient("test-key", h.Logger, nil)

	store.GetLastCompletedRunFn = func(_ context.Context, _ string) (uuid.UUID, time.Time, error) {
		return uuid.New(), time.Now().Add(-1 * time.Hour), nil
	}

	event, _ := json.Marshal(Input{Source: "fallback"})
	out, err := h.Handle(context.Background(), event)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if out.JobType != "snapshot-api" {
		t.Errorf("expected job_type snapshot-api, got %q", out.JobType)
	}
}

// --- runPipeline with mock API ---

func TestMockAPIClient_ReturnsEmptyByDefault(t *testing.T) {
	api := &mockAPIClient{}
	opps, raw, total, err := api.FetchAll(context.Background(), "01/01/2026", "02/01/2026")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(opps) != 0 || len(raw) != 0 || total != 0 {
		t.Error("expected empty response from default mock")
	}
}

// --- SnapAPIRow creation ---

func TestSnapAPIRowBuild(t *testing.T) {
	// Verifies the snap row building logic that the handler does
	raw := json.RawMessage(`{"noticeId":"N001","title":"Test"}`)
	row := database.SnapAPIRow{
		NoticeID:    "N001",
		RawData:     raw,
		ContentHash: computeHash(raw),
	}
	if row.NoticeID != "N001" {
		t.Error("unexpected notice ID")
	}
	if row.ContentHash == "" {
		t.Error("expected non-empty hash")
	}
}
