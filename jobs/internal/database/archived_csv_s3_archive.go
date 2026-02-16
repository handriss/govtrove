package database

import (
	"context"
	"time"

	"github.com/jackc/pgx/v5"
)

type ArchivedCSVDownloadLogRecord struct {
	ID                    int
	FiscalYear            int
	CheckedAt             time.Time
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

func (db *DB) InsertArchivedCSVDownloadLog(ctx context.Context, r *ArchivedCSVDownloadLogRecord) (int, error) {
	query := `
		INSERT INTO archived_csv_s3_archive_log (
			fiscal_year, result, http_status, etag, last_modified, content_length,
			file_size_bytes, compressed_size_bytes, row_count, sha256_hash,
			s3_key, download_duration_ms, compression_duration_ms, upload_duration_ms,
			error_message
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15)
		RETURNING id
	`

	var id int
	err := db.pool.QueryRow(ctx, query,
		r.FiscalYear, r.Result, r.HTTPStatus, r.ETag, r.LastModified, r.ContentLength,
		r.FileSizeBytes, r.CompressedSizeBytes, r.RowCount, r.SHA256Hash,
		r.S3Key, r.DownloadDurationMs, r.CompressionDurationMs, r.UploadDurationMs,
		r.ErrorMessage,
	).Scan(&id)
	return id, err
}

func (db *DB) GetLatestArchivedCSVETag(ctx context.Context, fiscalYear int) (string, error) {
	query := `
		SELECT etag FROM archived_csv_s3_archive_log
		WHERE fiscal_year = $1 AND result != 'error' AND etag IS NOT NULL
		ORDER BY checked_at DESC LIMIT 1
	`
	var etag string
	err := db.pool.QueryRow(ctx, query, fiscalYear).Scan(&etag)
	if err == pgx.ErrNoRows {
		return "", nil
	}
	return etag, err
}

func (db *DB) GetLatestArchivedCSVHash(ctx context.Context, fiscalYear int) (string, error) {
	query := `
		SELECT sha256_hash FROM archived_csv_s3_archive_log
		WHERE fiscal_year = $1 AND result = 'new_file' AND sha256_hash IS NOT NULL
		AND s3_key IS NOT NULL AND s3_key != ''
		ORDER BY checked_at DESC LIMIT 1
	`
	var hash string
	err := db.pool.QueryRow(ctx, query, fiscalYear).Scan(&hash)
	if err == pgx.ErrNoRows {
		return "", nil
	}
	return hash, err
}
