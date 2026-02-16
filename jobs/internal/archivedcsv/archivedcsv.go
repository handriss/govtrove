package archivedcsv

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
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/handriss/govtrove/jobs/internal/config"
	"github.com/handriss/govtrove/jobs/internal/database"
)

const (
	archivedCSVURL = "https://sam.gov/api/prod/fileextractservices/v1/api/download/Contract%20Opportunities/Archived%20Data/FY2026_archived_opportunities.csv"
	fiscalYear     = 2026

	rangeRequestTimeout = 30 * time.Second
	downloadTimeout     = 10 * time.Minute
	maxRetries          = 3
)

type Result struct {
	Outcome  string // etag_match, hash_match, new_file, not_found, error
	FileSize int64
	RowCount int
	Duration time.Duration
}

func Run(ctx context.Context, cfg *config.Config, db *database.DB, s3Client *s3.Client, logger *slog.Logger) (*Result, error) {
	result := &Result{}
	startTime := time.Now()

	storedETag, err := db.GetLatestArchivedCSVETag(ctx, fiscalYear)
	if err != nil {
		logger.Warn("failed to load stored ETag, proceeding without", "error", err)
	}

	logger.Info("checking archived CSV",
		"fiscal_year", fiscalYear,
		"has_stored_etag", storedETag != "",
	)

	// Range request to get ETag cheaply (1 byte)
	currentETag, contentLength, err := probeETag(ctx, logger)
	if err != nil {
		logError(ctx, db, logger, fiscalYear, err, nil)
		return nil, fmt.Errorf("ETag probe failed: %w", err)
	}

	if currentETag != "" && currentETag == storedETag {
		record := &database.ArchivedCSVDownloadLogRecord{
			FiscalYear:    fiscalYear,
			Result:        "etag_match",
			ETag:          strPtr(currentETag),
			ContentLength: &contentLength,
		}
		if _, err := db.InsertArchivedCSVDownloadLog(ctx, record); err != nil {
			logger.Warn("failed to log etag_match", "error", err)
		}
		logger.Info("archived CSV unchanged (ETag match)", "etag", currentETag)
		result.Outcome = "etag_match"
		result.Duration = time.Since(startTime)
		return result, nil
	}

	// ETag changed or first run — full download
	logger.Info("ETag changed or new, downloading full file",
		"stored_etag", storedETag,
		"current_etag", currentETag,
		"content_length", contentLength,
	)

	fileSize, hashHex, rowCount, tmpPath, downloadDur, err := downloadWithRetry(ctx, logger)
	if err != nil {
		logError(ctx, db, logger, fiscalYear, err, nil)
		return nil, err
	}
	defer os.Remove(tmpPath)

	httpStatus := http.StatusOK

	// Compare SHA-256 with previous
	prevHash, err := db.GetLatestArchivedCSVHash(ctx, fiscalYear)
	if err != nil {
		logger.Warn("failed to get previous hash", "error", err)
	}
	if prevHash != "" && prevHash == hashHex {
		record := &database.ArchivedCSVDownloadLogRecord{
			FiscalYear:         fiscalYear,
			Result:             "hash_match",
			HTTPStatus:         &httpStatus,
			ETag:               strPtr(currentETag),
			ContentLength:      &contentLength,
			FileSizeBytes:      &fileSize,
			RowCount:           &rowCount,
			SHA256Hash:         &hashHex,
			DownloadDurationMs: &downloadDur,
		}
		if _, err := db.InsertArchivedCSVDownloadLog(ctx, record); err != nil {
			logger.Warn("failed to log hash_match", "error", err)
		}
		logger.Info("archived CSV hash matches previous, skipping archive")
		result.Outcome = "hash_match"
		result.FileSize = fileSize
		result.Duration = time.Since(startTime)
		return result, nil
	}

	// Gzip compress
	compressStart := time.Now()
	gzPath, compressedSize, err := compressFile(tmpPath)
	if err != nil {
		logError(ctx, db, logger, fiscalYear, fmt.Errorf("compression failed: %w", err), &httpStatus)
		return nil, err
	}
	defer os.Remove(gzPath)
	compressionDur := int(time.Since(compressStart).Milliseconds())

	logger.Info("archived CSV compressed",
		"compressed_size_bytes", compressedSize,
		"ratio", fmt.Sprintf("%.1f%%", float64(compressedSize)/float64(fileSize)*100),
		"compression_ms", compressionDur,
	)

	// S3 upload
	today := time.Now().UTC().Format("2006-01-02")
	s3Key := fmt.Sprintf("raw/archived-csv/FY%d/%s.csv.gz", fiscalYear, today)
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
			logError(ctx, db, logger, fiscalYear, fmt.Errorf("S3 upload failed: %w", err), &httpStatus)
			return nil, fmt.Errorf("S3 upload failed: %w", err)
		}
		uploadDur = int(time.Since(uploadStart).Milliseconds())
		logger.Info("archived CSV uploaded to S3", "key", s3Key, "upload_ms", uploadDur)
	} else {
		logger.Info("S3 archive disabled, skipping upload", "would_be_key", s3Key)
		s3Key = ""
	}

	record := &database.ArchivedCSVDownloadLogRecord{
		FiscalYear:            fiscalYear,
		Result:                "new_file",
		HTTPStatus:            &httpStatus,
		ETag:                  strPtr(currentETag),
		ContentLength:         &contentLength,
		FileSizeBytes:         &fileSize,
		CompressedSizeBytes:   &compressedSize,
		RowCount:              &rowCount,
		SHA256Hash:            &hashHex,
		S3Key:                 strPtr(s3Key),
		DownloadDurationMs:    &downloadDur,
		CompressionDurationMs: &compressionDur,
		UploadDurationMs:      &uploadDur,
	}
	if _, err := db.InsertArchivedCSVDownloadLog(ctx, record); err != nil {
		logger.Warn("failed to log new_file", "error", err)
	}

	logger.Info("archived CSV archive complete",
		"result", "new_file",
		"fiscal_year", fiscalYear,
		"file_size_bytes", fileSize,
		"compressed_size_bytes", compressedSize,
		"row_count", rowCount,
		"s3_key", s3Key,
		"total_ms", time.Since(startTime).Milliseconds(),
	)

	result.Outcome = "new_file"
	result.FileSize = fileSize
	result.RowCount = rowCount
	result.Duration = time.Since(startTime)
	return result, nil
}

// probeETag does a 1-byte range request to get ETag and Content-Length without downloading the full file.
// SAM.gov redirects to a presigned S3 URL which supports range requests.
func probeETag(ctx context.Context, logger *slog.Logger) (etag string, contentLength int64, err error) {
	ctx, cancel := context.WithTimeout(ctx, rangeRequestTimeout)
	defer cancel()

	client := &http.Client{
		// Don't follow redirects automatically — we need to follow manually
		// because the presigned S3 URL supports range but the SAM.gov endpoint may not pass it through
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			return http.ErrUseLastResponse
		},
	}

	// First request to get the redirect URL
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, archivedCSVURL, nil)
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
		return "", 0, fmt.Errorf("archived CSV not found (404)")
	}

	// Follow redirect with range header
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

	// Parse Content-Range for total size: "bytes 0-0/TOTAL"
	if cr := resp2.Header.Get("Content-Range"); cr != "" {
		var start, end, total int64
		if n, _ := fmt.Sscanf(cr, "bytes %d-%d/%d", &start, &end, &total); n == 3 {
			contentLength = total
		}
	}

	logger.Debug("ETag probe result", "etag", etag, "content_length", contentLength)
	return etag, contentLength, nil
}

func downloadWithRetry(ctx context.Context, logger *slog.Logger) (fileSize int64, hashHex string, rowCount int, tmpPath string, downloadDurMs int, err error) {
	var lastErr error
	for attempt := 1; attempt <= maxRetries; attempt++ {
		fileSize, hashHex, rowCount, tmpPath, downloadDurMs, err = downloadFull(ctx, logger)
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

func downloadFull(ctx context.Context, logger *slog.Logger) (fileSize int64, hashHex string, rowCount int, tmpPath string, downloadDurMs int, err error) {
	dlCtx, cancel := context.WithTimeout(ctx, downloadTimeout)
	defer cancel()

	// SAM.gov redirects, so use default client (follows redirects)
	req, err := http.NewRequestWithContext(dlCtx, http.MethodGet, archivedCSVURL, nil)
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
		return 0, "", 0, "", 0, fmt.Errorf("archived CSV not found (404)")
	}
	if resp.StatusCode != http.StatusOK {
		return 0, "", 0, "", 0, fmt.Errorf("unexpected HTTP status: %d", resp.StatusCode)
	}

	tmpFile, err := os.CreateTemp("", "govtrove-archived-csv-*.csv")
	if err != nil {
		return 0, "", 0, "", 0, fmt.Errorf("create temp file: %w", err)
	}
	tmpPath = tmpFile.Name()

	hasher := sha256.New()
	lineCounter := &lineCountWriter{}
	multiWriter := io.MultiWriter(tmpFile, hasher, lineCounter)

	fileSize, err = io.Copy(multiWriter, resp.Body)
	tmpFile.Close()
	if err != nil {
		os.Remove(tmpPath)
		return 0, "", 0, "", 0, fmt.Errorf("download stream failed: %w", err)
	}

	downloadDurMs = int(time.Since(downloadStart).Milliseconds())
	hashHex = hex.EncodeToString(hasher.Sum(nil))
	rowCount = lineCounter.lines

	logger.Info("archived CSV downloaded",
		"file_size_bytes", fileSize,
		"row_count", rowCount,
		"sha256", hashHex,
		"download_ms", downloadDurMs,
	)

	return fileSize, hashHex, rowCount, tmpPath, downloadDurMs, nil
}

func compressFile(srcPath string) (gzPath string, compressedSize int64, err error) {
	src, err := os.Open(srcPath)
	if err != nil {
		return "", 0, fmt.Errorf("open source: %w", err)
	}
	defer src.Close()

	gzFile, err := os.CreateTemp("", "govtrove-archived-csv-*.csv.gz")
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

func logError(ctx context.Context, db *database.DB, logger *slog.Logger, fy int, err error, httpStatus *int) {
	errMsg := err.Error()
	record := &database.ArchivedCSVDownloadLogRecord{
		FiscalYear:   fy,
		Result:       "error",
		HTTPStatus:   httpStatus,
		ErrorMessage: &errMsg,
	}
	if _, dbErr := db.InsertArchivedCSVDownloadLog(ctx, record); dbErr != nil {
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
