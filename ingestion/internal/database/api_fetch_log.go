package database

import (
	"context"
	"time"

	"github.com/jackc/pgx/v5"
)

type APIFetchLogRecord struct {
	ID                    int
	FetchedAt             time.Time
	Result                string
	PostedFrom            *string
	TotalRecords          *int
	RecordsFetched        *int
	PagesFetched          *int
	FileSizeBytes         *int64
	CompressedSizeBytes   *int64
	SHA256Hash            *string
	S3Key                 *string
	FetchDurationMs       *int
	CompressionDurationMs *int
	UploadDurationMs      *int
	ErrorMessage          *string
}

func (db *DB) InsertAPIFetchLog(ctx context.Context, r *APIFetchLogRecord) (int, error) {
	query := `
		INSERT INTO api_fetch_log (
			result, posted_from, total_records, records_fetched, pages_fetched,
			file_size_bytes, compressed_size_bytes, sha256_hash, s3_key,
			fetch_duration_ms, compression_duration_ms, upload_duration_ms,
			error_message
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13)
		RETURNING id
	`

	var id int
	err := db.pool.QueryRow(ctx, query,
		r.Result, r.PostedFrom, r.TotalRecords, r.RecordsFetched, r.PagesFetched,
		r.FileSizeBytes, r.CompressedSizeBytes, r.SHA256Hash, r.S3Key,
		r.FetchDurationMs, r.CompressionDurationMs, r.UploadDurationMs,
		r.ErrorMessage,
	).Scan(&id)
	return id, err
}

// GetLatestAPIFetchHash returns the SHA-256 from the most recent new_file entry
// that was actually uploaded to S3.
func (db *DB) GetLatestAPIFetchHash(ctx context.Context) (string, error) {
	query := `
		SELECT sha256_hash FROM api_fetch_log
		WHERE result = 'new_file' AND sha256_hash IS NOT NULL
		AND s3_key IS NOT NULL AND s3_key != ''
		ORDER BY fetched_at DESC LIMIT 1
	`
	var hash string
	err := db.pool.QueryRow(ctx, query).Scan(&hash)
	if err == pgx.ErrNoRows {
		return "", nil
	}
	return hash, err
}

// HasAPIFetchForDate checks if a new_file result with S3 upload exists for the given UTC date.
func (db *DB) HasAPIFetchForDate(ctx context.Context, date time.Time) (bool, error) {
	startOfDay := time.Date(date.Year(), date.Month(), date.Day(), 0, 0, 0, 0, time.UTC)
	endOfDay := startOfDay.Add(24 * time.Hour)

	query := `
		SELECT EXISTS(
			SELECT 1 FROM api_fetch_log
			WHERE result = 'new_file'
			AND s3_key IS NOT NULL AND s3_key != ''
			AND fetched_at >= $1 AND fetched_at < $2
		)
	`
	var exists bool
	err := db.pool.QueryRow(ctx, query, startOfDay, endOfDay).Scan(&exists)
	return exists, err
}
