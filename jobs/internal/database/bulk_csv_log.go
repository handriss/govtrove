package database

import (
	"context"

	"github.com/jackc/pgx/v5"
)

type BulkCSVLogRecord struct {
	ID                    int
	Source                string
	Result                string
	HTTPStatus            *int
	ETag                  *string
	LastModified          *string
	ContentLength         *int64
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

func (db *DB) InsertBulkCSVLog(ctx context.Context, r *BulkCSVLogRecord) (int, error) {
	query := `
		INSERT INTO pipeline.bulk_csv_log (
			source, result, http_status, etag, last_modified, content_length,
			file_size_bytes, compressed_size_bytes, row_count, sha256_hash,
			s3_key, download_duration_ms, compression_duration_ms, upload_duration_ms,
			error_message
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15)
		RETURNING id
	`

	var id int
	err := db.pool.QueryRow(ctx, query,
		r.Source, r.Result, r.HTTPStatus, r.ETag, r.LastModified, r.ContentLength,
		r.FileSizeBytes, r.CompressedSizeBytes, r.RowCount, r.SHA256Hash,
		r.S3Key, r.DownloadDurationMs, r.CompressionDurationMs, r.UploadDurationMs,
		r.ErrorMessage,
	).Scan(&id)
	return id, err
}

func (db *DB) GetLatestBulkCSVHash(ctx context.Context, source string) (string, error) {
	query := `
		SELECT sha256_hash FROM pipeline.bulk_csv_log
		WHERE source = $1 AND result = 'new_file'
		AND sha256_hash IS NOT NULL
		AND s3_key IS NOT NULL AND s3_key != ''
		ORDER BY checked_at DESC LIMIT 1
	`
	var hash string
	err := db.pool.QueryRow(ctx, query, source).Scan(&hash)
	if err == pgx.ErrNoRows {
		return "", nil
	}
	return hash, err
}

func (db *DB) GetLatestBulkCSVHeaders(ctx context.Context, source string) (etag, lastModified string, err error) {
	query := `
		SELECT etag, last_modified FROM pipeline.bulk_csv_log
		WHERE source = $1 AND result = 'new_file'
		AND (etag IS NOT NULL OR last_modified IS NOT NULL)
		ORDER BY checked_at DESC LIMIT 1
	`
	var e, lm *string
	err = db.pool.QueryRow(ctx, query, source).Scan(&e, &lm)
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

func (db *DB) GetLatestBulkCSVS3Key(ctx context.Context, source string) (string, error) {
	query := `
		SELECT s3_key FROM pipeline.bulk_csv_log
		WHERE source = $1 AND result = 'new_file'
		AND s3_key IS NOT NULL AND s3_key != ''
		ORDER BY checked_at DESC LIMIT 1
	`
	var key string
	err := db.pool.QueryRow(ctx, query, source).Scan(&key)
	if err == pgx.ErrNoRows {
		return "", nil
	}
	return key, err
}
