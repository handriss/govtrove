package apiprobe

import (
	"compress/gzip"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/url"
	"os"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/handriss/govtrove/ingestion/internal/config"
	"github.com/handriss/govtrove/ingestion/internal/database"
	"github.com/handriss/govtrove/ingestion/internal/samgov"
)

type ArchiveResult struct {
	Outcome        string // new_file, hash_match, already_archived, error
	RecordsFetched int
	FileSize       int64
	Duration       time.Duration
}

func RunArchive(ctx context.Context, cfg *config.APIProbeConfig, db *database.DB, s3Client *s3.Client, logger *slog.Logger) (*ArchiveResult, error) {
	today := time.Now().UTC()
	result := &ArchiveResult{}
	startTime := time.Now()

	// Idempotency: skip if we already archived today
	alreadyDone, err := db.HasAPIFetchForDate(ctx, today)
	if err != nil {
		return nil, fmt.Errorf("idempotency check failed: %w", err)
	}
	if alreadyDone {
		logger.Info("already archived API data for today, skipping")
		record := &database.APIFetchLogRecord{Result: "already_archived"}
		if _, err := db.InsertAPIFetchLog(ctx, record); err != nil {
			logger.Warn("failed to log already_archived", "error", err)
		}
		result.Outcome = "already_archived"
		result.Duration = time.Since(startTime)
		return result, nil
	}

	et, err := time.LoadLocation("America/New_York")
	if err != nil {
		return nil, fmt.Errorf("load ET timezone: %w", err)
	}

	now := time.Now().In(et)
	postedFrom := now.AddDate(0, 0, -cfg.LookbackDays).Format("01/02/2006")
	postedTo := now.Format("01/02/2006")
	logger.Info("fetching API data", "posted_from", postedFrom, "posted_to", postedTo, "lookback_days", cfg.LookbackDays)

	// Stream paginated results to a temp JSONL file, computing SHA-256 as we go
	tmpFile, err := os.CreateTemp("", "govtrove-api-*.jsonl")
	if err != nil {
		return nil, fmt.Errorf("create temp file: %w", err)
	}
	defer os.Remove(tmpFile.Name())
	defer tmpFile.Close()

	hasher := sha256.New()
	writer := io.MultiWriter(tmpFile, hasher)

	httpClient := samgov.NewTrackedHTTPClient(db, logger)
	offset := 0
	totalRecords := 0
	recordsFetched := 0
	pagesFetched := 0
	fetchStart := time.Now()

	for {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		default:
		}

		u, err := url.Parse(samgov.SearchAPIBaseURL)
		if err != nil {
			return nil, fmt.Errorf("parse base URL: %w", err)
		}

		q := u.Query()
		q.Set("api_key", cfg.SAMAPIKey)
		q.Set("postedFrom", postedFrom)
		q.Set("postedTo", postedTo)
		q.Set("limit", fmt.Sprintf("%d", samgov.DefaultPageLimit))
		q.Set("offset", fmt.Sprintf("%d", offset))
		u.RawQuery = q.Encode()

		resp, err := httpClient.Get(ctx, u.String())
		if err != nil {
			logFetchError(ctx, db, logger, fmt.Errorf("page %d request failed: %w", pagesFetched, err), postedFrom)
			return nil, fmt.Errorf("API request failed at page %d: %w", pagesFetched, err)
		}

		if resp.StatusCode != 200 {
			resp.Body.Close()
			logFetchError(ctx, db, logger, fmt.Errorf("page %d HTTP %d", pagesFetched, resp.StatusCode), postedFrom)
			return nil, fmt.Errorf("API returned status %d at page %d", resp.StatusCode, pagesFetched)
		}

		// Read the raw response body for archival
		rawBody, err := io.ReadAll(resp.Body)
		resp.Body.Close()
		if err != nil {
			logFetchError(ctx, db, logger, fmt.Errorf("read body page %d: %w", pagesFetched, err), postedFrom)
			return nil, fmt.Errorf("read response body at page %d: %w", pagesFetched, err)
		}

		// Write raw JSON as one JSONL line
		if _, err := writer.Write(rawBody); err != nil {
			return nil, fmt.Errorf("write page %d: %w", pagesFetched, err)
		}
		if _, err := writer.Write([]byte("\n")); err != nil {
			return nil, fmt.Errorf("write newline page %d: %w", pagesFetched, err)
		}

		// Parse to get totalRecords and record count for this page
		var searchResp struct {
			TotalRecords      int               `json:"totalRecords"`
			OpportunitiesData []json.RawMessage  `json:"opportunitiesData"`
		}
		if err := json.Unmarshal(rawBody, &searchResp); err != nil {
			logFetchError(ctx, db, logger, fmt.Errorf("parse page %d: %w", pagesFetched, err), postedFrom)
			return nil, fmt.Errorf("parse page %d: %w", pagesFetched, err)
		}

		pageCount := len(searchResp.OpportunitiesData)
		if pagesFetched == 0 {
			totalRecords = searchResp.TotalRecords
			logger.Info("API reports total records", "total", totalRecords)
		}

		recordsFetched += pageCount
		pagesFetched++

		logger.Debug("fetched page",
			"page", pagesFetched,
			"offset", offset,
			"records_on_page", pageCount,
			"total_fetched", recordsFetched,
		)

		if pageCount < samgov.DefaultPageLimit {
			break
		}
		offset += samgov.DefaultPageLimit

		// Respect rate limits
		time.Sleep(samgov.PaginationDelay)
	}

	fetchDur := int(time.Since(fetchStart).Milliseconds())
	hashHex := hex.EncodeToString(hasher.Sum(nil))

	fileInfo, err := tmpFile.Stat()
	if err != nil {
		return nil, fmt.Errorf("stat temp file: %w", err)
	}
	fileSize := fileInfo.Size()

	logger.Info("API fetch complete",
		"total_records", totalRecords,
		"records_fetched", recordsFetched,
		"pages", pagesFetched,
		"file_size_bytes", fileSize,
		"sha256", hashHex,
		"fetch_ms", fetchDur,
	)

	// SHA-256 dedup
	prevHash, err := db.GetLatestAPIFetchHash(ctx)
	if err != nil {
		logger.Warn("failed to get previous hash", "error", err)
	}
	if prevHash != "" && prevHash == hashHex {
		record := &database.APIFetchLogRecord{
			Result:          "hash_match",
			PostedFrom:      &postedFrom,
			TotalRecords:    &totalRecords,
			RecordsFetched:  &recordsFetched,
			PagesFetched:    &pagesFetched,
			FileSizeBytes:   &fileSize,
			SHA256Hash:      &hashHex,
			FetchDurationMs: &fetchDur,
		}
		if _, err := db.InsertAPIFetchLog(ctx, record); err != nil {
			logger.Warn("failed to log hash_match", "error", err)
		}
		logger.Info("API data hash matches previous, skipping archive")
		result.Outcome = "hash_match"
		result.RecordsFetched = recordsFetched
		result.FileSize = fileSize
		result.Duration = time.Since(startTime)
		return result, nil
	}

	// Gzip compress
	compressStart := time.Now()
	gzFile, err := os.CreateTemp("", "govtrove-api-*.jsonl.gz")
	if err != nil {
		return nil, fmt.Errorf("create gzip temp file: %w", err)
	}
	defer os.Remove(gzFile.Name())
	defer gzFile.Close()

	if _, err := tmpFile.Seek(0, io.SeekStart); err != nil {
		return nil, fmt.Errorf("seek temp file: %w", err)
	}

	gzWriter := gzip.NewWriter(gzFile)
	if _, err := io.Copy(gzWriter, tmpFile); err != nil {
		return nil, fmt.Errorf("gzip compression failed: %w", err)
	}
	if err := gzWriter.Close(); err != nil {
		return nil, fmt.Errorf("gzip close failed: %w", err)
	}

	gzInfo, err := gzFile.Stat()
	if err != nil {
		return nil, fmt.Errorf("stat gzip file: %w", err)
	}
	compressedSize := gzInfo.Size()
	compressionDur := int(time.Since(compressStart).Milliseconds())

	logger.Info("API data compressed",
		"compressed_size_bytes", compressedSize,
		"ratio", fmt.Sprintf("%.1f%%", float64(compressedSize)/float64(fileSize)*100),
		"compression_ms", compressionDur,
	)

	// S3 upload
	s3Key := fmt.Sprintf("raw/api/%s.jsonl.gz", today.Format("2006-01-02"))
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
			logFetchError(ctx, db, logger, fmt.Errorf("S3 upload failed: %w", err), postedFrom)
			return nil, fmt.Errorf("S3 upload failed: %w", err)
		}
		uploadDur = int(time.Since(uploadStart).Milliseconds())
		logger.Info("API data uploaded to S3", "key", s3Key, "upload_ms", uploadDur)
	} else {
		logger.Info("S3 archive disabled, skipping upload", "would_be_key", s3Key)
		s3Key = ""
	}

	// Log new_file
	record := &database.APIFetchLogRecord{
		Result:                "new_file",
		PostedFrom:            &postedFrom,
		TotalRecords:          &totalRecords,
		RecordsFetched:        &recordsFetched,
		PagesFetched:          &pagesFetched,
		FileSizeBytes:         &fileSize,
		CompressedSizeBytes:   &compressedSize,
		SHA256Hash:            &hashHex,
		S3Key:                 strPtr(s3Key),
		FetchDurationMs:       &fetchDur,
		CompressionDurationMs: &compressionDur,
		UploadDurationMs:      &uploadDur,
	}
	if _, err := db.InsertAPIFetchLog(ctx, record); err != nil {
		logger.Warn("failed to log new_file", "error", err)
	}

	logger.Info("API archive complete",
		"result", "new_file",
		"records_fetched", recordsFetched,
		"file_size_bytes", fileSize,
		"compressed_size_bytes", compressedSize,
		"s3_key", s3Key,
		"total_ms", time.Since(startTime).Milliseconds(),
	)

	result.Outcome = "new_file"
	result.RecordsFetched = recordsFetched
	result.FileSize = fileSize
	result.Duration = time.Since(startTime)
	return result, nil
}

func logFetchError(ctx context.Context, db *database.DB, logger *slog.Logger, err error, postedFrom string) {
	errMsg := err.Error()
	record := &database.APIFetchLogRecord{
		Result:       "error",
		PostedFrom:   &postedFrom,
		ErrorMessage: &errMsg,
	}
	if _, dbErr := db.InsertAPIFetchLog(ctx, record); dbErr != nil {
		logger.Warn("failed to log error", "error", dbErr)
	}
}

func strPtr(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}
