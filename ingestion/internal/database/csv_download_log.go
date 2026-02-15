package database

import (
	"context"
	"time"

	"github.com/jackc/pgx/v5"
)

type CSVDownloadLogRecord struct {
	ID                    int
	CheckedAt             time.Time
	Result                string
	HTTPStatus            *int
	ETag                  *string
	LastModified          *string
	FileSizeBytes         *int64
	CompressedSizeBytes   *int64
	RowCount              *int
	SHA256Hash            *string
	S3Key                 *string
	DownloadDurationMs    *int
	CompressionDurationMs *int
	UploadDurationMs      *int
	ErrorMessage          *string
}

func (db *DB) InsertCSVDownloadLog(ctx context.Context, r *CSVDownloadLogRecord) (int, error) {
	query := `
		INSERT INTO csv_download_log (
			result, http_status, etag, last_modified,
			file_size_bytes, compressed_size_bytes, row_count, sha256_hash,
			s3_key, download_duration_ms, compression_duration_ms, upload_duration_ms,
			error_message
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13)
		RETURNING id
	`

	var id int
	err := db.pool.QueryRow(ctx, query,
		r.Result, r.HTTPStatus, r.ETag, r.LastModified,
		r.FileSizeBytes, r.CompressedSizeBytes, r.RowCount, r.SHA256Hash,
		r.S3Key, r.DownloadDurationMs, r.CompressionDurationMs, r.UploadDurationMs,
		r.ErrorMessage,
	).Scan(&id)
	return id, err
}

// GetLatestCSVDownloadHash returns the SHA-256 hash from the most recent 'new_file' log entry
// that was actually uploaded to S3. Local runs (S3 disabled) don't count.
func (db *DB) GetLatestCSVDownloadHash(ctx context.Context) (string, error) {
	query := `
		SELECT sha256_hash FROM csv_download_log
		WHERE result = 'new_file' AND sha256_hash IS NOT NULL
		AND s3_key IS NOT NULL AND s3_key != ''
		ORDER BY checked_at DESC LIMIT 1
	`
	var hash string
	err := db.pool.QueryRow(ctx, query).Scan(&hash)
	if err == pgx.ErrNoRows {
		return "", nil
	}
	return hash, err
}

// GetLatestCSVDownloadHeaders returns the ETag and Last-Modified from the most recent non-error log entry.
func (db *DB) GetLatestCSVDownloadHeaders(ctx context.Context) (etag, lastModified string, err error) {
	query := `
		SELECT etag, last_modified FROM csv_download_log
		WHERE result != 'error' AND (etag IS NOT NULL OR last_modified IS NOT NULL)
		ORDER BY checked_at DESC LIMIT 1
	`
	var e, lm *string
	err = db.pool.QueryRow(ctx, query).Scan(&e, &lm)
	if err == pgx.ErrNoRows {
		return "", "", nil
	}
	if err != nil {
		return "", "", err
	}
	if e != nil {
		etag = *e
	}
	if lm != nil {
		lastModified = *lm
	}
	return etag, lastModified, nil
}

// HasNewFileForDate checks if a 'new_file' result with a successful S3 upload
// already exists for the given UTC date. Local runs (S3 disabled) don't count.
func (db *DB) HasNewFileForDate(ctx context.Context, date time.Time) (bool, error) {
	startOfDay := time.Date(date.Year(), date.Month(), date.Day(), 0, 0, 0, 0, time.UTC)
	endOfDay := startOfDay.Add(24 * time.Hour)

	query := `
		SELECT EXISTS(
			SELECT 1 FROM csv_download_log
			WHERE result = 'new_file'
			AND s3_key IS NOT NULL AND s3_key != ''
			AND checked_at >= $1 AND checked_at < $2
		)
	`
	var exists bool
	err := db.pool.QueryRow(ctx, query, startOfDay, endOfDay).Scan(&exists)
	return exists, err
}
