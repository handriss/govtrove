package database

import (
	"context"
	"time"
)

type CSVCacheHeaders struct {
	ID               int
	URL              string
	ETag             string
	LastModified     string
	ContentLength    int64
	LastCheckedAt    time.Time
	LastDownloadedAt *time.Time
	CreatedAt        time.Time
}

// GetCSVCacheHeaders retrieves cached headers for a URL
func (db *DB) GetCSVCacheHeaders(ctx context.Context, url string) (*CSVCacheHeaders, error) {
	query := `
		SELECT id, url, etag, last_modified, content_length, last_checked_at, last_downloaded_at, created_at
		FROM csv_cache_headers
		WHERE url = $1
	`

	var h CSVCacheHeaders
	var etag, lastModified *string
	var contentLength *int64

	err := db.pool.QueryRow(ctx, query, url).Scan(
		&h.ID,
		&h.URL,
		&etag,
		&lastModified,
		&contentLength,
		&h.LastCheckedAt,
		&h.LastDownloadedAt,
		&h.CreatedAt,
	)
	if err != nil {
		return nil, err
	}

	if etag != nil {
		h.ETag = *etag
	}
	if lastModified != nil {
		h.LastModified = *lastModified
	}
	if contentLength != nil {
		h.ContentLength = *contentLength
	}

	return &h, nil
}

// UpsertCSVCacheHeaders inserts or updates cache headers for a URL
func (db *DB) UpsertCSVCacheHeaders(ctx context.Context, h *CSVCacheHeaders) error {
	query := `
		INSERT INTO csv_cache_headers (url, etag, last_modified, content_length, last_checked_at, last_downloaded_at)
		VALUES ($1, $2, $3, $4, NOW(), $5)
		ON CONFLICT (url) DO UPDATE SET
			etag = EXCLUDED.etag,
			last_modified = EXCLUDED.last_modified,
			content_length = EXCLUDED.content_length,
			last_checked_at = NOW(),
			last_downloaded_at = EXCLUDED.last_downloaded_at
	`

	var etag, lastModified *string
	var contentLength *int64

	if h.ETag != "" {
		etag = &h.ETag
	}
	if h.LastModified != "" {
		lastModified = &h.LastModified
	}
	if h.ContentLength > 0 {
		contentLength = &h.ContentLength
	}

	_, err := db.pool.Exec(ctx, query, h.URL, etag, lastModified, contentLength, h.LastDownloadedAt)
	return err
}

// UpdateCSVCacheLastChecked updates just the last_checked_at timestamp
func (db *DB) UpdateCSVCacheLastChecked(ctx context.Context, url string) error {
	query := `UPDATE csv_cache_headers SET last_checked_at = NOW() WHERE url = $1`
	_, err := db.pool.Exec(ctx, query, url)
	return err
}
