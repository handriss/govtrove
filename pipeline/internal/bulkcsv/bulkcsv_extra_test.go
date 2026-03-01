package bulkcsv

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/handriss/govtrove/pipeline/internal/config"
	"github.com/handriss/govtrove/pipeline/internal/database"
)

// --- streamToTemp ---

func TestStreamToTemp_BasicContent(t *testing.T) {
	body := strings.NewReader("line1\nline2\nline3\n")
	fileSize, hashHex, rowCount, tmpPath, err := streamToTemp(body)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	defer os.Remove(tmpPath)

	if fileSize != 18 {
		t.Errorf("expected file size 18, got %d", fileSize)
	}
	if rowCount != 3 {
		t.Errorf("expected 3 rows, got %d", rowCount)
	}
	if hashHex == "" {
		t.Error("expected non-empty hash")
	}
	if tmpPath == "" {
		t.Error("expected non-empty temp path")
	}
}

func TestStreamToTemp_EmptyBody(t *testing.T) {
	body := strings.NewReader("")
	fileSize, _, rowCount, tmpPath, err := streamToTemp(body)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	defer os.Remove(tmpPath)

	if fileSize != 0 {
		t.Errorf("expected 0 bytes, got %d", fileSize)
	}
	if rowCount != 0 {
		t.Errorf("expected 0 rows, got %d", rowCount)
	}
}

func TestStreamToTemp_DeterministicHash(t *testing.T) {
	content := "same content\n"
	_, hash1, _, path1, _ := streamToTemp(strings.NewReader(content))
	defer os.Remove(path1)
	_, hash2, _, path2, _ := streamToTemp(strings.NewReader(content))
	defer os.Remove(path2)

	if hash1 != hash2 {
		t.Errorf("hashes should match: %s != %s", hash1, hash2)
	}
}

func TestStreamToTemp_DifferentContentDifferentHash(t *testing.T) {
	_, hash1, _, path1, _ := streamToTemp(strings.NewReader("content A\n"))
	defer os.Remove(path1)
	_, hash2, _, path2, _ := streamToTemp(strings.NewReader("content B\n"))
	defer os.Remove(path2)

	if hash1 == hash2 {
		t.Error("different content should produce different hashes")
	}
}

func TestStreamToTemp_NoTrailingNewline(t *testing.T) {
	body := strings.NewReader("line1\nline2")
	_, _, rowCount, tmpPath, err := streamToTemp(body)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	defer os.Remove(tmpPath)

	if rowCount != 1 {
		t.Errorf("expected 1 row (only counted newlines), got %d", rowCount)
	}
}

// --- compressFile ---

func TestCompressFile_ValidFile(t *testing.T) {
	// Write a temp file
	tmp, err := os.CreateTemp("", "compress-test-*.csv")
	if err != nil {
		t.Fatal(err)
	}
	content := "NoticeId,Title\nID-001,Test Opportunity\n"
	tmp.WriteString(content)
	tmp.Close()
	defer os.Remove(tmp.Name())

	gzPath, compressedSize, err := compressFile(tmp.Name())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	defer os.Remove(gzPath)

	if compressedSize <= 0 {
		t.Errorf("expected positive compressed size, got %d", compressedSize)
	}
	if !strings.HasSuffix(gzPath, ".csv.gz") {
		t.Errorf("expected .csv.gz suffix, got %s", gzPath)
	}
}

func TestCompressFile_NonexistentFile(t *testing.T) {
	_, _, err := compressFile("/nonexistent/path/file.csv")
	if err == nil {
		t.Error("expected error for nonexistent file")
	}
}

func TestCompressFile_EmptyFile(t *testing.T) {
	tmp, err := os.CreateTemp("", "compress-empty-*.csv")
	if err != nil {
		t.Fatal(err)
	}
	tmp.Close()
	defer os.Remove(tmp.Name())

	gzPath, compressedSize, err := compressFile(tmp.Name())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	defer os.Remove(gzPath)

	// Even empty gzip has a header
	if compressedSize <= 0 {
		t.Errorf("expected positive size for gzip header, got %d", compressedSize)
	}
}

// --- lineCountWriter ---

func TestLineCountWriter_MultipleLines(t *testing.T) {
	w := &lineCountWriter{}
	data := []byte("line1\nline2\nline3\n")
	n, err := w.Write(data)
	if err != nil {
		t.Fatal(err)
	}
	if n != len(data) {
		t.Errorf("expected %d bytes, got %d", len(data), n)
	}
	if w.lines != 3 {
		t.Errorf("expected 3 lines, got %d", w.lines)
	}
}

func TestLineCountWriter_NoNewlines(t *testing.T) {
	w := &lineCountWriter{}
	w.Write([]byte("no newlines"))
	if w.lines != 0 {
		t.Errorf("expected 0 lines, got %d", w.lines)
	}
}

func TestLineCountWriter_EmptyWrite(t *testing.T) {
	w := &lineCountWriter{}
	n, err := w.Write([]byte{})
	if err != nil {
		t.Fatal(err)
	}
	if n != 0 {
		t.Errorf("expected 0 bytes, got %d", n)
	}
	if w.lines != 0 {
		t.Errorf("expected 0 lines, got %d", w.lines)
	}
}

func TestLineCountWriter_MultipleWrites(t *testing.T) {
	w := &lineCountWriter{}
	w.Write([]byte("a\n"))
	w.Write([]byte("b\n"))
	w.Write([]byte("c\n"))
	if w.lines != 3 {
		t.Errorf("expected 3 lines, got %d", w.lines)
	}
}

func TestLineCountWriter_ConsecutiveNewlines(t *testing.T) {
	w := &lineCountWriter{}
	w.Write([]byte("\n\n\n"))
	if w.lines != 3 {
		t.Errorf("expected 3 lines, got %d", w.lines)
	}
}

// --- strPtr ---

func TestStrPtr_Empty(t *testing.T) {
	got := strPtr("")
	if got != nil {
		t.Error("expected nil for empty string")
	}
}

func TestStrPtr_NonEmpty(t *testing.T) {
	got := strPtr("hello")
	if got == nil || *got != "hello" {
		t.Error("expected pointer to 'hello'")
	}
}

// --- FormatSummary ---

func TestFormatSummary_Empty(t *testing.T) {
	summary := FormatSummary(nil)
	if summary != "" {
		t.Errorf("expected empty summary, got %q", summary)
	}
}

func TestFormatSummary_MixedOutcomes(t *testing.T) {
	results := []SourceResult{
		{Source: "active", Outcome: "new_file", Size: 5242880, Rows: 10000},
		{Source: "bad", Outcome: "error"},
	}
	summary := FormatSummary(results)
	if !strings.Contains(summary, "active: new_file") {
		t.Error("missing active new_file in summary")
	}
	if !strings.Contains(summary, "10000 rows") {
		t.Error("missing row count")
	}
	if !strings.Contains(summary, "bad: error") {
		t.Error("missing bad error")
	}
}

func TestFormatSummary_HashMatch(t *testing.T) {
	results := []SourceResult{
		{Source: "active", Outcome: "hash_match"},
	}
	summary := FormatSummary(results)
	if !strings.Contains(summary, "hash_match") {
		t.Error("missing hash_match in summary")
	}
	// hash_match should NOT show MB/rows
	if strings.Contains(summary, "MB") {
		t.Error("hash_match should not show file size")
	}
}

// --- downloadFull ---

func TestDownloadFull_404(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	defer server.Close()

	log := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelWarn}))
	_, _, _, _, _, err := downloadFull(context.Background(), log, server.URL)
	if err == nil {
		t.Error("expected error for 404")
	}
	if !strings.Contains(err.Error(), "404") {
		t.Errorf("expected 404 in error, got: %v", err)
	}
}

func TestDownloadFull_403(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusForbidden)
	}))
	defer server.Close()

	log := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelWarn}))
	_, _, _, _, _, err := downloadFull(context.Background(), log, server.URL)
	if err == nil {
		t.Error("expected error for 403")
	}
	if !strings.Contains(err.Error(), "403") {
		t.Errorf("expected 403 in error, got: %v", err)
	}
}

func TestDownloadFull_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		fmt.Fprint(w, "col1,col2\nval1,val2\n")
	}))
	defer server.Close()

	log := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelWarn}))
	fileSize, hash, rowCount, tmpPath, durMs, err := downloadFull(context.Background(), log, server.URL)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	defer os.Remove(tmpPath)

	if fileSize <= 0 {
		t.Error("expected positive file size")
	}
	if hash == "" {
		t.Error("expected non-empty hash")
	}
	if rowCount != 2 {
		t.Errorf("expected 2 rows, got %d", rowCount)
	}
	if durMs < 0 {
		t.Error("expected non-negative duration")
	}
}

// --- downloadWithRetry ---

func TestDownloadWithRetry_SuccessOnFirstAttempt(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		fmt.Fprint(w, "data\n")
	}))
	defer server.Close()

	log := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelWarn}))
	_, _, _, tmpPath, _, err := downloadWithRetry(context.Background(), log, server.URL)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	defer os.Remove(tmpPath)
}

func TestDownloadWithRetry_AllAttemptsFail(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()

	log := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelWarn}))
	_, _, _, _, _, err := downloadWithRetry(context.Background(), log, server.URL)
	if err == nil {
		t.Error("expected error after all retries fail")
	}
	if !strings.Contains(err.Error(), "3 attempts") {
		t.Errorf("expected '3 attempts' in error, got: %v", err)
	}
}

func TestDownloadWithRetry_SuccessOnSecondAttempt(t *testing.T) {
	attempts := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		attempts++
		if attempts < 2 {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusOK)
		fmt.Fprint(w, "data\n")
	}))
	defer server.Close()

	log := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelWarn}))
	_, _, _, tmpPath, _, err := downloadWithRetry(context.Background(), log, server.URL)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	defer os.Remove(tmpPath)

	if attempts != 2 {
		t.Errorf("expected 2 attempts, got %d", attempts)
	}
}

// --- S3 existence check clearing state ---

func TestS3MissingFileClearsState(t *testing.T) {
	csvBody := "NoticeId,Title\nID-001,Test\n"
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// No conditional GET support — always return the file
		w.WriteHeader(http.StatusOK)
		fmt.Fprint(w, csvBody)
	}))
	defer server.Close()

	origSrcs := Sources
	defer func() { Sources = origSrcs }()
	Sources = []Source{{Key: "test-src", URL: server.URL, S3Prefix: "raw/csv"}}

	store := &mockBulkStore{
		// Return a previous hash (would normally trigger hash_match)
		hashFn: func(_ context.Context, _ string) (string, error) {
			return "some-old-hash", nil
		},
		// Return a previous S3 key
		s3KeyFn: func(_ context.Context, _ string) (string, error) {
			return "raw/csv/2026-01-01.csv.gz", nil
		},
	}

	// S3 HeadObject fails → file missing → state should be cleared → file re-downloaded
	s3c := &mockS3Client{
		headFn: func(_ context.Context, _ *s3.HeadObjectInput, _ ...func(*s3.Options)) (*s3.HeadObjectOutput, error) {
			return nil, fmt.Errorf("not found")
		},
	}

	cfg := &config.Config{
		S3Bucket:         "test-bucket",
		S3ArchiveEnabled: true,
	}

	results, err := Run(context.Background(), cfg, store, s3c, slog.Default())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(results))
	}
	// Should be new_file (not hash_match) because S3 missing cleared the previous hash
	if results[0].Outcome != "new_file" {
		t.Errorf("expected new_file (S3 missing should force re-download), got %q", results[0].Outcome)
	}
}

// --- logError ---

func TestLogError_InsertsRecord(t *testing.T) {
	var logged *database.BulkCSVLogRecord
	store := &mockBulkStore{
		logFn: func(_ context.Context, r *database.BulkCSVLogRecord) (int, error) {
			logged = r
			return 1, nil
		},
	}

	log := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelWarn}))
	status := 500
	logError(context.Background(), store, log, "test-source", fmt.Errorf("test error"), &status)

	if logged == nil {
		t.Fatal("expected log record")
	}
	if logged.Source != "test-source" {
		t.Errorf("expected source test-source, got %q", logged.Source)
	}
	if logged.Result != "error" {
		t.Errorf("expected result error, got %q", logged.Result)
	}
	if logged.HTTPStatus == nil || *logged.HTTPStatus != 500 {
		t.Error("expected HTTP status 500")
	}
	if logged.ErrorMessage == nil || *logged.ErrorMessage != "test error" {
		t.Error("expected error message")
	}
}

func TestLogError_NilHTTPStatus(t *testing.T) {
	var logged *database.BulkCSVLogRecord
	store := &mockBulkStore{
		logFn: func(_ context.Context, r *database.BulkCSVLogRecord) (int, error) {
			logged = r
			return 1, nil
		},
	}

	log := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelWarn}))
	logError(context.Background(), store, log, "src", fmt.Errorf("network error"), nil)

	if logged == nil {
		t.Fatal("expected log record")
	}
	if logged.HTTPStatus != nil {
		t.Error("expected nil HTTP status")
	}
}

// --- conditional GET with If-Modified-Since ---

func TestConditionalGet_IfModifiedSince(t *testing.T) {
	var receivedIfModifiedSince string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		receivedIfModifiedSince = r.Header.Get("If-Modified-Since")
		if receivedIfModifiedSince != "" {
			w.WriteHeader(http.StatusNotModified)
			return
		}
		w.WriteHeader(http.StatusOK)
		fmt.Fprint(w, "data\n")
	}))
	defer server.Close()

	origSrcs := Sources
	defer func() { Sources = origSrcs }()
	Sources = []Source{{Key: "test-src", URL: server.URL, S3Prefix: "raw/csv"}}

	store := &mockBulkStore{
		headersFn: func(_ context.Context, _ string) (string, string, error) {
			return "", "Wed, 01 Jan 2026 00:00:00 GMT", nil
		},
	}
	cfg := &config.Config{S3ArchiveEnabled: false}

	results, err := Run(context.Background(), cfg, store, &mockS3Client{}, slog.Default())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if results[0].Outcome != "not_modified" {
		t.Errorf("expected not_modified, got %q", results[0].Outcome)
	}
	if receivedIfModifiedSince != "Wed, 01 Jan 2026 00:00:00 GMT" {
		t.Errorf("expected If-Modified-Since header, got %q", receivedIfModifiedSince)
	}
}

// --- conditional GET with both headers ---

func TestConditionalGet_BothHeaders(t *testing.T) {
	var receivedETag, receivedLM string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		receivedETag = r.Header.Get("If-None-Match")
		receivedLM = r.Header.Get("If-Modified-Since")
		if receivedETag != "" {
			w.WriteHeader(http.StatusNotModified)
			return
		}
		w.WriteHeader(http.StatusOK)
		fmt.Fprint(w, "data\n")
	}))
	defer server.Close()

	origSrcs := Sources
	defer func() { Sources = origSrcs }()
	Sources = []Source{{Key: "test-src", URL: server.URL, S3Prefix: "raw/csv"}}

	store := &mockBulkStore{
		headersFn: func(_ context.Context, _ string) (string, string, error) {
			return `"etag-abc"`, "Wed, 01 Jan 2026 00:00:00 GMT", nil
		},
	}
	cfg := &config.Config{S3ArchiveEnabled: false}

	results, err := Run(context.Background(), cfg, store, &mockS3Client{}, slog.Default())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if results[0].Outcome != "not_modified" {
		t.Errorf("expected not_modified, got %q", results[0].Outcome)
	}
	if receivedETag != `"etag-abc"` {
		t.Errorf("expected ETag header, got %q", receivedETag)
	}
	if receivedLM != "Wed, 01 Jan 2026 00:00:00 GMT" {
		t.Errorf("expected Last-Modified header, got %q", receivedLM)
	}
}

// --- S3 upload failure ---

func TestS3UploadFailure(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		fmt.Fprint(w, "NoticeId,Title\nID-001,Test\n")
	}))
	defer server.Close()

	origSrcs := Sources
	defer func() { Sources = origSrcs }()
	Sources = []Source{{Key: "test-src", URL: server.URL, S3Prefix: "raw/csv"}}

	store := &mockBulkStore{}
	s3c := &mockS3Client{
		putFn: func(_ context.Context, _ *s3.PutObjectInput, _ ...func(*s3.Options)) (*s3.PutObjectOutput, error) {
			return nil, fmt.Errorf("S3 access denied")
		},
	}
	cfg := &config.Config{
		S3Bucket:         "test-bucket",
		S3ArchiveEnabled: true,
	}

	results, err := Run(context.Background(), cfg, store, s3c, slog.Default())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// Source should fail with error outcome
	if len(results) != 1 || results[0].Outcome != "error" {
		t.Errorf("expected error outcome, got %v", results)
	}
}

// --- ETag and headers from response are recorded ---

func TestNewFileRecordsETagAndLastModified(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("ETag", `"response-etag"`)
		w.Header().Set("Last-Modified", "Thu, 01 Jan 2026 12:00:00 GMT")
		w.WriteHeader(http.StatusOK)
		fmt.Fprint(w, "col1\nval1\n")
	}))
	defer server.Close()

	origSrcs := Sources
	defer func() { Sources = origSrcs }()
	Sources = []Source{{Key: "test-src", URL: server.URL, S3Prefix: "raw/csv"}}

	var loggedRecord *database.BulkCSVLogRecord
	store := &mockBulkStore{
		logFn: func(_ context.Context, r *database.BulkCSVLogRecord) (int, error) {
			if r.Result == "new_file" {
				loggedRecord = r
			}
			return 1, nil
		},
	}
	cfg := &config.Config{S3ArchiveEnabled: false}

	results, err := Run(context.Background(), cfg, store, &mockS3Client{}, slog.Default())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if results[0].Outcome != "new_file" {
		t.Fatalf("expected new_file, got %q", results[0].Outcome)
	}
	if loggedRecord == nil {
		t.Fatal("expected log record for new_file")
	}
	if loggedRecord.ETag == nil || *loggedRecord.ETag != `"response-etag"` {
		t.Errorf("expected ETag recorded, got %v", loggedRecord.ETag)
	}
	if loggedRecord.LastModified == nil || *loggedRecord.LastModified != "Thu, 01 Jan 2026 12:00:00 GMT" {
		t.Errorf("expected Last-Modified recorded, got %v", loggedRecord.LastModified)
	}
}
