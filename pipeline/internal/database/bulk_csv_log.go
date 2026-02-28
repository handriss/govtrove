package database

import (
	"context"
	"fmt"

	"github.com/google/uuid"
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
	PipelineRunID         *uuid.UUID
	Status                *string
}

func (db *DB) InsertBulkCSVLog(ctx context.Context, r *BulkCSVLogRecord) (int, error) {
	query := `
		INSERT INTO pipeline.bulk_csv_log (
			source, result, http_status, etag, last_modified, content_length,
			file_size_bytes, compressed_size_bytes, row_count, sha256_hash,
			s3_key, download_duration_ms, compression_duration_ms, upload_duration_ms,
			error_message, pipeline_run_id
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16)
		RETURNING id
	`

	var id int
	err := db.pool.QueryRow(ctx, query,
		r.Source, r.Result, r.HTTPStatus, r.ETag, r.LastModified, r.ContentLength,
		r.FileSizeBytes, r.CompressedSizeBytes, r.RowCount, r.SHA256Hash,
		r.S3Key, r.DownloadDurationMs, r.CompressionDurationMs, r.UploadDurationMs,
		r.ErrorMessage, r.PipelineRunID,
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

func (db *DB) GetBulkCSVLogByS3Key(ctx context.Context, s3Key string) (*BulkCSVLogRecord, error) {
	var r BulkCSVLogRecord
	err := db.pool.QueryRow(ctx, `
		SELECT id, source, result, s3_key, row_count, pipeline_run_id, status
		FROM pipeline.bulk_csv_log
		WHERE s3_key = $1 AND result = 'new_file'
		ORDER BY checked_at DESC LIMIT 1
	`, s3Key).Scan(&r.ID, &r.Source, &r.Result, &r.S3Key, &r.RowCount, &r.PipelineRunID, &r.Status)
	if err == pgx.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &r, nil
}

func (db *DB) UpdateBulkCSVLogIngestion(ctx context.Context, id int, ingestionRunID uuid.UUID, recordCount int, status string) error {
	_, err := db.pool.Exec(ctx, `
		UPDATE pipeline.bulk_csv_log
		SET ingestion_run_id = $2, record_count = $3, status = $4
		WHERE id = $1
	`, id, ingestionRunID, recordCount, status)
	if err != nil {
		return fmt.Errorf("update bulk_csv_log ingestion: %w", err)
	}
	return nil
}
