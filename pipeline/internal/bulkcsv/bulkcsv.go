package bulkcsv

import (
	"compress/gzip"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/handriss/govtrove/pipeline/internal/config"
	"github.com/handriss/govtrove/pipeline/internal/database"
)

// BulkCSVStore is the subset of database.Store used by the bulkcsv package.
type BulkCSVStore interface {
	InsertBulkCSVLog(ctx context.Context, r *database.BulkCSVLogRecord) (int, error)
	GetLatestBulkCSVHash(ctx context.Context, source string) (string, error)
	GetLatestBulkCSVHeaders(ctx context.Context, source string) (string, string, error)
	GetLatestBulkCSVS3Key(ctx context.Context, source string) (string, error)
}

const (
	rangeRequestTimeout = 30 * time.Second
	downloadTimeout     = 10 * time.Minute
	maxRetries          = 3
)

type SourceResult struct {
	Source   string
	Outcome string // new_file, not_modified, hash_match, error
	Size    int64
	Rows    int
	S3Key   string
}

func Run(ctx context.Context, cfg *config.Config, db BulkCSVStore, s3Client S3Client, logger *slog.Logger) ([]SourceResult, error) {
	var results []SourceResult

	for _, src := range Sources {
		srcLogger := logger.With("source", src.Key)
		srcLogger.Info("processing source")

		r, err := processSource(ctx, cfg, db, s3Client, srcLogger, src)
		if err != nil {
			srcLogger.Error("source failed", "error", err)
			results = append(results, SourceResult{Source: src.Key, Outcome: "error"})
			continue
		}

		results = append(results, *r)
		srcLogger.Info("source done", "outcome", r.Outcome)
	}

	return results, nil
}

func processSource(ctx context.Context, cfg *config.Config, db BulkCSVStore, s3Client S3Client, logger *slog.Logger, src Source) (*SourceResult, error) {
	startTime := time.Now()

	// Load previous state
	prevHash, err := db.GetLatestBulkCSVHash(ctx, src.Key)
	if err != nil {
		logger.Warn("failed to get previous hash", "error", err)
	}

	prevETag, prevLastModified, err := db.GetLatestBulkCSVHeaders(ctx, src.Key)
	if err != nil {
		logger.Warn("failed to get previous headers", "error", err)
	}

	// S3 existence check: if we have a previous s3_key, verify it still exists
	if s3Client != nil && cfg.S3ArchiveEnabled {
		lastS3Key, keyErr := db.GetLatestBulkCSVS3Key(ctx, src.Key)
		if keyErr != nil {
			logger.Warn("failed to get latest S3 key", "error", keyErr)
		}
		if lastS3Key != "" {
			_, headErr := s3Client.HeadObject(ctx, &s3.HeadObjectInput{
				Bucket: aws.String(cfg.S3Bucket),
				Key:    aws.String(lastS3Key),
			})
			if headErr != nil {
				logger.Warn("previous S3 file missing, will re-download", "key", lastS3Key)
				prevHash = ""
				prevETag = ""
				prevLastModified = ""
			}
		}
	}

	logger.Info("checking for changes",
		"has_etag", prevETag != "",
		"has_last_modified", prevLastModified != "",
		"has_hash", prevHash != "",
	)

	// Check for changes using the appropriate strategy
	if src.UseRangeProbe {
		return processWithRangeProbe(ctx, cfg, db, s3Client, logger, src, prevETag, prevHash, startTime)
	}
	return processWithConditionalGet(ctx, cfg, db, s3Client, logger, src, prevETag, prevLastModified, prevHash, startTime)
}

func processWithConditionalGet(
	ctx context.Context, cfg *config.Config, db BulkCSVStore, s3Client S3Client,
	logger *slog.Logger, src Source,
	prevETag, prevLastModified, prevHash string,
	startTime time.Time,
) (*SourceResult, error) {
	httpClient := &http.Client{Timeout: downloadTimeout}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, src.URL, nil)
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}
	if prevETag != "" {
		req.Header.Set("If-None-Match", prevETag)
	}
	if prevLastModified != "" {
		req.Header.Set("If-Modified-Since", prevLastModified)
	}

	downloadStart := time.Now()
	resp, err := httpClient.Do(req)
	if err != nil {
		logError(ctx, db, logger, src.Key, err, nil)
		return nil, fmt.Errorf("HTTP request failed: %w", err)
	}
	defer resp.Body.Close()

	httpStatus := resp.StatusCode

	if resp.StatusCode == http.StatusNotModified {
		logger.Info("not modified (304)")
		return &SourceResult{Source: src.Key, Outcome: "not_modified"}, nil
	}

	if resp.StatusCode != http.StatusOK {
		err := fmt.Errorf("unexpected HTTP status: %d", resp.StatusCode)
		logError(ctx, db, logger, src.Key, err, &httpStatus)
		return nil, err
	}

	respETag := resp.Header.Get("ETag")
	respLastModified := resp.Header.Get("Last-Modified")

	// Stream to temp file
	fileSize, hashHex, rowCount, tmpPath, err := streamToTemp(resp.Body)
	if err != nil {
		logError(ctx, db, logger, src.Key, fmt.Errorf("download failed: %w", err), &httpStatus)
		return nil, err
	}
	defer os.Remove(tmpPath)
	downloadDur := int(time.Since(downloadStart).Milliseconds())

	logger.Info("downloaded",
		"file_size_bytes", fileSize,
		"row_count", rowCount,
		"sha256", hashHex,
		"download_ms", downloadDur,
	)

	if prevHash != "" && prevHash == hashHex {
		logger.Info("hash matches previous, skipping")
		return &SourceResult{Source: src.Key, Outcome: "hash_match"}, nil
	}

	return archiveFile(ctx, cfg, db, s3Client, logger, src, tmpPath, fileSize, hashHex, rowCount,
		&httpStatus, respETag, respLastModified, 0, downloadDur, startTime)
}

func processWithRangeProbe(
	ctx context.Context, cfg *config.Config, db BulkCSVStore, s3Client S3Client,
	logger *slog.Logger, src Source,
	prevETag, prevHash string,
	startTime time.Time,
) (*SourceResult, error) {
	currentETag, contentLength, err := probeETag(ctx, logger, src.URL)
	if err != nil {
		logError(ctx, db, logger, src.Key, fmt.Errorf("ETag probe failed: %w", err), nil)
		return nil, fmt.Errorf("ETag probe failed: %w", err)
	}

	if currentETag != "" && currentETag == prevETag {
		logger.Info("unchanged (ETag match)", "etag", currentETag)
		return &SourceResult{Source: src.Key, Outcome: "not_modified"}, nil
	}

	logger.Info("ETag changed or new, downloading",
		"stored_etag", prevETag,
		"current_etag", currentETag,
		"content_length", contentLength,
	)

	fileSize, hashHex, rowCount, tmpPath, downloadDur, err := downloadWithRetry(ctx, logger, src.URL)
	if err != nil {
		logError(ctx, db, logger, src.Key, err, nil)
		return nil, err
	}
	defer os.Remove(tmpPath)

	httpStatus := http.StatusOK

	if prevHash != "" && prevHash == hashHex {
		logger.Info("hash matches previous, skipping")
		return &SourceResult{Source: src.Key, Outcome: "hash_match"}, nil
	}

	return archiveFile(ctx, cfg, db, s3Client, logger, src, tmpPath, fileSize, hashHex, rowCount,
		&httpStatus, currentETag, "", contentLength, downloadDur, startTime)
}

func archiveFile(
	ctx context.Context, cfg *config.Config, db BulkCSVStore, s3Client S3Client,
	logger *slog.Logger, src Source,
	tmpPath string, fileSize int64, hashHex string, rowCount int,
	httpStatus *int, etag, lastModified string, contentLength int64,
	downloadDur int, startTime time.Time,
) (*SourceResult, error) {
	// Gzip compress
	compressStart := time.Now()
	gzPath, compressedSize, err := compressFile(tmpPath)
	if err != nil {
		logError(ctx, db, logger, src.Key, fmt.Errorf("compression failed: %w", err), httpStatus)
		return nil, err
	}
	defer os.Remove(gzPath)
	compressionDur := int(time.Since(compressStart).Milliseconds())

	logger.Info("compressed",
		"compressed_size_bytes", compressedSize,
		"ratio", fmt.Sprintf("%.1f%%", float64(compressedSize)/float64(fileSize)*100),
		"compression_ms", compressionDur,
	)

	// S3 upload
	today := time.Now().UTC().Format("2006-01-02")
	s3Key := fmt.Sprintf("%s/%s.csv.gz", src.S3Prefix, today)
	var uploadDur int

	if cfg.S3ArchiveEnabled && s3Client != nil {
		gzFile, err := os.Open(gzPath)
		if err != nil {
			return nil, fmt.Errorf("open gzip file: %w", err)
		}
		defer gzFile.Close()

		uploadStart := time.Now()
		_, err = s3Client.PutObject(ctx, &s3.PutObjectInput{
			Bucket:      aws.String(cfg.S3Bucket),
			Key:         aws.String(s3Key),
			Body:        gzFile,
			ContentType: aws.String("application/gzip"),
		})
		if err != nil {
			logError(ctx, db, logger, src.Key, fmt.Errorf("S3 upload failed: %w", err), httpStatus)
			return nil, fmt.Errorf("S3 upload failed: %w", err)
		}
		uploadDur = int(time.Since(uploadStart).Milliseconds())
		logger.Info("uploaded to S3", "key", s3Key, "upload_ms", uploadDur)
	} else {
		logger.Info("S3 archive disabled, skipping upload", "would_be_key", s3Key)
		s3Key = ""
	}

	// Log new_file to DB
	var cl *int64
	if contentLength > 0 {
		cl = &contentLength
	}
	record := &database.BulkCSVLogRecord{
		Source:                src.Key,
		Result:                "new_file",
		HTTPStatus:            httpStatus,
		ETag:                  strPtr(etag),
		LastModified:          strPtr(lastModified),
		ContentLength:         cl,
		FileSizeBytes:         &fileSize,
		CompressedSizeBytes:   &compressedSize,
		RowCount:              &rowCount,
		SHA256Hash:            &hashHex,
		S3Key:                 strPtr(s3Key),
		DownloadDurationMs:    &downloadDur,
		CompressionDurationMs: &compressionDur,
		UploadDurationMs:      &uploadDur,
	}
	if _, err := db.InsertBulkCSVLog(ctx, record); err != nil {
		logger.Warn("failed to log new_file", "error", err)
	}

	logger.Info("archive complete",
		"result", "new_file",
		"file_size_bytes", fileSize,
		"compressed_size_bytes", compressedSize,
		"row_count", rowCount,
		"s3_key", s3Key,
		"total_ms", time.Since(startTime).Milliseconds(),
	)

	return &SourceResult{
		Source:  src.Key,
		Outcome: "new_file",
		Size:   fileSize,
		Rows:   rowCount,
		S3Key:  s3Key,
	}, nil
}

// probeETag does a 1-byte range request to get ETag and Content-Length without downloading.
// SAM.gov redirects to a presigned S3 URL which supports range requests.
func probeETag(ctx context.Context, logger *slog.Logger, url string) (etag string, contentLength int64, err error) {
	ctx, cancel := context.WithTimeout(ctx, rangeRequestTimeout)
	defer cancel()

	client := &http.Client{
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			return http.ErrUseLastResponse
		},
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return "", 0, fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("Range", "bytes=0-0")

	resp, err := client.Do(req)
	if err != nil {
		return "", 0, fmt.Errorf("range request: %w", err)
	}
	resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		return "", 0, fmt.Errorf("file not found (404)")
	}

	redirectURL := resp.Header.Get("Location")
	if redirectURL == "" {
		return "", 0, fmt.Errorf("no redirect location (status %d)", resp.StatusCode)
	}

	req2, err := http.NewRequestWithContext(ctx, http.MethodGet, redirectURL, nil)
	if err != nil {
		return "", 0, fmt.Errorf("create redirect request: %w", err)
	}
	req2.Header.Set("Range", "bytes=0-0")

	resp2, err := http.DefaultClient.Do(req2)
	if err != nil {
		return "", 0, fmt.Errorf("redirect range request: %w", err)
	}
	resp2.Body.Close()

	if resp2.StatusCode != http.StatusPartialContent {
		return "", 0, fmt.Errorf("unexpected status from range request: %d", resp2.StatusCode)
	}

	etag = resp2.Header.Get("ETag")
	contentLength = resp2.ContentLength

	if cr := resp2.Header.Get("Content-Range"); cr != "" {
		var start, end, total int64
		if n, _ := fmt.Sscanf(cr, "bytes %d-%d/%d", &start, &end, &total); n == 3 {
			contentLength = total
		}
	}

	logger.Debug("ETag probe result", "etag", etag, "content_length", contentLength)
	return etag, contentLength, nil
}

func downloadWithRetry(ctx context.Context, logger *slog.Logger, url string) (fileSize int64, hashHex string, rowCount int, tmpPath string, downloadDurMs int, err error) {
	var lastErr error
	for attempt := 1; attempt <= maxRetries; attempt++ {
		fileSize, hashHex, rowCount, tmpPath, downloadDurMs, err = downloadFull(ctx, logger, url)
		if err == nil {
			return
		}
		lastErr = err
		if attempt < maxRetries {
			backoff := time.Duration(1<<(attempt-1)) * time.Second
			logger.Warn("download failed, retrying", "attempt", attempt, "backoff", backoff, "error", err)
			select {
			case <-ctx.Done():
				return 0, "", 0, "", 0, ctx.Err()
			case <-time.After(backoff):
			}
		}
	}
	return 0, "", 0, "", 0, fmt.Errorf("download failed after %d attempts: %w", maxRetries, lastErr)
}

func downloadFull(ctx context.Context, logger *slog.Logger, url string) (fileSize int64, hashHex string, rowCount int, tmpPath string, downloadDurMs int, err error) {
	dlCtx, cancel := context.WithTimeout(ctx, downloadTimeout)
	defer cancel()

	req, err := http.NewRequestWithContext(dlCtx, http.MethodGet, url, nil)
	if err != nil {
		return 0, "", 0, "", 0, fmt.Errorf("create request: %w", err)
	}

	downloadStart := time.Now()
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return 0, "", 0, "", 0, fmt.Errorf("HTTP request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		return 0, "", 0, "", 0, fmt.Errorf("file not found (404)")
	}
	if resp.StatusCode != http.StatusOK {
		return 0, "", 0, "", 0, fmt.Errorf("unexpected HTTP status: %d", resp.StatusCode)
	}

	fileSize, hashHex, rowCount, tmpPath, err = streamToTemp(resp.Body)
	if err != nil {
		return 0, "", 0, "", 0, err
	}
	downloadDurMs = int(time.Since(downloadStart).Milliseconds())

	logger.Info("downloaded",
		"file_size_bytes", fileSize,
		"row_count", rowCount,
		"sha256", hashHex,
		"download_ms", downloadDurMs,
	)

	return fileSize, hashHex, rowCount, tmpPath, downloadDurMs, nil
}

func streamToTemp(body io.Reader) (fileSize int64, hashHex string, rowCount int, tmpPath string, err error) {
	tmpFile, err := os.CreateTemp("", "govtrove-csv-*.csv")
	if err != nil {
		return 0, "", 0, "", fmt.Errorf("create temp file: %w", err)
	}
	tmpPath = tmpFile.Name()

	hasher := sha256.New()
	lineCounter := &lineCountWriter{}
	multiWriter := io.MultiWriter(tmpFile, hasher, lineCounter)

	fileSize, err = io.Copy(multiWriter, body)
	tmpFile.Close()
	if err != nil {
		os.Remove(tmpPath)
		return 0, "", 0, "", fmt.Errorf("download stream failed: %w", err)
	}

	hashHex = hex.EncodeToString(hasher.Sum(nil))
	rowCount = lineCounter.lines
	return fileSize, hashHex, rowCount, tmpPath, nil
}

func compressFile(srcPath string) (gzPath string, compressedSize int64, err error) {
	src, err := os.Open(srcPath)
	if err != nil {
		return "", 0, fmt.Errorf("open source: %w", err)
	}
	defer src.Close()

	gzFile, err := os.CreateTemp("", "govtrove-csv-*.csv.gz")
	if err != nil {
		return "", 0, fmt.Errorf("create gzip temp file: %w", err)
	}
	gzPath = gzFile.Name()

	gzWriter := gzip.NewWriter(gzFile)
	if _, err := io.Copy(gzWriter, src); err != nil {
		gzFile.Close()
		os.Remove(gzPath)
		return "", 0, fmt.Errorf("gzip write: %w", err)
	}
	if err := gzWriter.Close(); err != nil {
		gzFile.Close()
		os.Remove(gzPath)
		return "", 0, fmt.Errorf("gzip close: %w", err)
	}

	info, err := gzFile.Stat()
	gzFile.Close()
	if err != nil {
		os.Remove(gzPath)
		return "", 0, fmt.Errorf("stat gzip: %w", err)
	}

	return gzPath, info.Size(), nil
}

func logError(ctx context.Context, db BulkCSVStore, logger *slog.Logger, source string, err error, httpStatus *int) {
	errMsg := err.Error()
	record := &database.BulkCSVLogRecord{
		Source:       source,
		Result:       "error",
		HTTPStatus:   httpStatus,
		ErrorMessage: &errMsg,
	}
	if _, dbErr := db.InsertBulkCSVLog(ctx, record); dbErr != nil {
		logger.Warn("failed to log error", "error", dbErr)
	}
}

func strPtr(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}

type lineCountWriter struct {
	lines int
}

func (w *lineCountWriter) Write(p []byte) (int, error) {
	for _, b := range p {
		if b == '\n' {
			w.lines++
		}
	}
	return len(p), nil
}

// FormatSummary builds a human-readable summary of all source results.
func FormatSummary(results []SourceResult) string {
	var b strings.Builder
	for _, r := range results {
		fmt.Fprintf(&b, "  %s: %s", r.Source, r.Outcome)
		if r.Outcome == "new_file" {
			fmt.Fprintf(&b, " (%.1f MB, %d rows)", float64(r.Size)/1024/1024, r.Rows)
		}
		b.WriteString("\n")
	}
	return b.String()
}
