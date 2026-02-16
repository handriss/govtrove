package csvarchive

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
	"github.com/handriss/govtrove/jobs/internal/samgov"
)

type Result struct {
	Outcome  string // cache_hit, hash_match, new_file, already_archived, error
	FileSize int64
	Duration time.Duration
}

func Run(ctx context.Context, cfg *config.Config, db *database.DB, s3Client *s3.Client, logger *slog.Logger) (*Result, error) {
	today := time.Now().UTC()
	result := &Result{}
	startTime := time.Now()

	// Idempotency: skip if we already archived a new file today
	alreadyDone, err := db.HasNewFileForDate(ctx, today)
	if err != nil {
		return nil, fmt.Errorf("idempotency check failed: %w", err)
	}
	if alreadyDone {
		logger.Info("already archived CSV for today, skipping")
		record := &database.CSVDownloadLogRecord{Result: "already_archived"}
		if _, err := db.InsertCSVDownloadLog(ctx, record); err != nil {
			logger.Warn("failed to log already_archived", "error", err)
		}
		result.Outcome = "already_archived"
		result.Duration = time.Since(startTime)
		return result, nil
	}

	// Load cached headers from previous runs
	etag, lastModified, err := db.GetLatestCSVDownloadHeaders(ctx)
	if err != nil {
		logger.Warn("failed to load cached headers, proceeding without conditional request", "error", err)
	}

	// Also check the shared csv_cache_headers table used by the ingestion service
	if etag == "" && lastModified == "" {
		cached, cacheErr := db.GetCSVCacheHeaders(ctx, samgov.FullCSVURL)
		if cacheErr == nil && cached != nil {
			etag = cached.ETag
			lastModified = cached.LastModified
		}
	}

	logger.Info("checking SAM.gov CSV",
		"has_etag", etag != "",
		"has_last_modified", lastModified != "",
	)

	// Download
	httpClient := &http.Client{Timeout: 10 * time.Minute}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, samgov.FullCSVURL, nil)
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}
	if etag != "" {
		req.Header.Set("If-None-Match", etag)
	}
	if lastModified != "" {
		req.Header.Set("If-Modified-Since", lastModified)
	}

	downloadStart := time.Now()
	resp, err := httpClient.Do(req)
	if err != nil {
		logError(ctx, db, logger, err, nil)
		return nil, fmt.Errorf("HTTP request failed: %w", err)
	}
	defer resp.Body.Close()

	respETag := resp.Header.Get("ETag")
	respLastModified := resp.Header.Get("Last-Modified")
	httpStatus := resp.StatusCode

	// 304 Not Modified
	if resp.StatusCode == http.StatusNotModified {
		downloadDur := int(time.Since(downloadStart).Milliseconds())
		record := &database.CSVDownloadLogRecord{
			Result:             "cache_hit",
			HTTPStatus:         &httpStatus,
			ETag:               strPtr(etag),
			LastModified:       strPtr(lastModified),
			DownloadDurationMs: &downloadDur,
		}
		if _, err := db.InsertCSVDownloadLog(ctx, record); err != nil {
			logger.Warn("failed to log cache_hit", "error", err)
		}
		logger.Info("CSV not modified (304)")
		result.Outcome = "cache_hit"
		result.Duration = time.Since(startTime)
		return result, nil
	}

	if resp.StatusCode != http.StatusOK {
		err := fmt.Errorf("unexpected HTTP status: %d", resp.StatusCode)
		logError(ctx, db, logger, err, &httpStatus)
		return nil, err
	}

	// Stream to temp file, computing SHA-256 and counting bytes/lines
	tmpFile, err := os.CreateTemp("", "govtrove-csv-*.csv")
	if err != nil {
		return nil, fmt.Errorf("create temp file: %w", err)
	}
	defer os.Remove(tmpFile.Name())
	defer tmpFile.Close()

	hasher := sha256.New()
	lineCounter := &lineCountWriter{}
	multiWriter := io.MultiWriter(tmpFile, hasher, lineCounter)

	fileSize, err := io.Copy(multiWriter, resp.Body)
	if err != nil {
		logError(ctx, db, logger, fmt.Errorf("download stream failed: %w", err), &httpStatus)
		return nil, fmt.Errorf("download stream failed: %w", err)
	}
	downloadDur := int(time.Since(downloadStart).Milliseconds())
	hashHex := hex.EncodeToString(hasher.Sum(nil))
	rowCount := lineCounter.lines

	logger.Info("CSV downloaded",
		"file_size_bytes", fileSize,
		"row_count", rowCount,
		"sha256", hashHex,
		"download_ms", downloadDur,
	)

	// Compare SHA-256 with previous
	prevHash, err := db.GetLatestCSVDownloadHash(ctx)
	if err != nil {
		logger.Warn("failed to get previous hash", "error", err)
	}
	if prevHash != "" && prevHash == hashHex {
		record := &database.CSVDownloadLogRecord{
			Result:             "hash_match",
			HTTPStatus:         &httpStatus,
			ETag:               strPtr(respETag),
			LastModified:       strPtr(respLastModified),
			FileSizeBytes:      &fileSize,
			RowCount:           &rowCount,
			SHA256Hash:         &hashHex,
			DownloadDurationMs: &downloadDur,
		}
		if _, err := db.InsertCSVDownloadLog(ctx, record); err != nil {
			logger.Warn("failed to log hash_match", "error", err)
		}
		logger.Info("CSV hash matches previous, skipping archive")
		result.Outcome = "hash_match"
		result.FileSize = fileSize
		result.Duration = time.Since(startTime)
		return result, nil
	}

	// Gzip compress
	compressStart := time.Now()
	gzFile, err := os.CreateTemp("", "govtrove-csv-*.csv.gz")
	if err != nil {
		return nil, fmt.Errorf("create gzip temp file: %w", err)
	}
	defer os.Remove(gzFile.Name())
	defer gzFile.Close()

	if _, err := tmpFile.Seek(0, io.SeekStart); err != nil {
		return nil, fmt.Errorf("seek temp file: %w", err)
	}

	gzWriter := gzip.NewWriter(gzFile)
	compressedSize, err := io.Copy(gzWriter, tmpFile)
	if err != nil {
		return nil, fmt.Errorf("gzip compression failed: %w", err)
	}
	if err := gzWriter.Close(); err != nil {
		return nil, fmt.Errorf("gzip close failed: %w", err)
	}

	// Get actual compressed file size (gzip writer reports input bytes, not output)
	gzInfo, err := gzFile.Stat()
	if err != nil {
		return nil, fmt.Errorf("stat gzip file: %w", err)
	}
	compressedSize = gzInfo.Size()
	compressionDur := int(time.Since(compressStart).Milliseconds())

	logger.Info("CSV compressed",
		"compressed_size_bytes", compressedSize,
		"ratio", fmt.Sprintf("%.1f%%", float64(compressedSize)/float64(fileSize)*100),
		"compression_ms", compressionDur,
	)

	// S3 upload
	s3Key := fmt.Sprintf("raw/csv/%s.csv.gz", today.Format("2006-01-02"))
	var uploadDur int
	if cfg.S3ArchiveEnabled && s3Client != nil {
		if _, err := gzFile.Seek(0, io.SeekStart); err != nil {
			return nil, fmt.Errorf("seek gzip file: %w", err)
		}

		uploadStart := time.Now()
		_, err := s3Client.PutObject(ctx, &s3.PutObjectInput{
			Bucket:      aws.String(cfg.S3Bucket),
			Key:         aws.String(s3Key),
			Body:        gzFile,
			ContentType: aws.String("application/gzip"),
		})
		if err != nil {
			logError(ctx, db, logger, fmt.Errorf("S3 upload failed: %w", err), &httpStatus)
			return nil, fmt.Errorf("S3 upload failed: %w", err)
		}
		uploadDur = int(time.Since(uploadStart).Milliseconds())
		logger.Info("CSV uploaded to S3", "key", s3Key, "upload_ms", uploadDur)
	} else {
		logger.Info("S3 archive disabled, skipping upload", "would_be_key", s3Key)
		s3Key = ""
	}

	// Update shared csv_cache_headers so the ingestion service benefits too
	if respETag != "" || respLastModified != "" {
		now := time.Now()
		cacheHeaders := &database.CSVCacheHeaders{
			URL:              samgov.FullCSVURL,
			ETag:             respETag,
			LastModified:     respLastModified,
			ContentLength:    fileSize,
			LastDownloadedAt: &now,
		}
		if err := db.UpsertCSVCacheHeaders(ctx, cacheHeaders); err != nil {
			logger.Warn("failed to update csv_cache_headers", "error", err)
		}
	}

	// Log new_file
	record := &database.CSVDownloadLogRecord{
		Result:                "new_file",
		HTTPStatus:            &httpStatus,
		ETag:                  strPtr(respETag),
		LastModified:          strPtr(respLastModified),
		FileSizeBytes:         &fileSize,
		CompressedSizeBytes:   &compressedSize,
		RowCount:              &rowCount,
		SHA256Hash:            &hashHex,
		S3Key:                 strPtr(s3Key),
		DownloadDurationMs:    &downloadDur,
		CompressionDurationMs: &compressionDur,
		UploadDurationMs:      &uploadDur,
	}
	if _, err := db.InsertCSVDownloadLog(ctx, record); err != nil {
		logger.Warn("failed to log new_file", "error", err)
	}

	logger.Info("CSV archive complete",
		"result", "new_file",
		"file_size_bytes", fileSize,
		"compressed_size_bytes", compressedSize,
		"row_count", rowCount,
		"s3_key", s3Key,
		"total_ms", time.Since(startTime).Milliseconds(),
	)

	result.Outcome = "new_file"
	result.FileSize = fileSize
	result.Duration = time.Since(startTime)
	return result, nil
}

func logError(ctx context.Context, db *database.DB, logger *slog.Logger, err error, httpStatus *int) {
	errMsg := err.Error()
	record := &database.CSVDownloadLogRecord{
		Result:       "error",
		HTTPStatus:   httpStatus,
		ErrorMessage: &errMsg,
	}
	if _, dbErr := db.InsertCSVDownloadLog(ctx, record); dbErr != nil {
		logger.Warn("failed to log error", "error", dbErr)
	}
}

func strPtr(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}

// lineCountWriter counts newline characters written through it.
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
