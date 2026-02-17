package database

import (
	"context"
	"encoding/json"
	"fmt"
	"time"
)

type SAMGovRequest struct {
	ID                int
	RequestTimestamp  time.Time
	Endpoint          string
	Method            string
	HTTPStatusCode    *int
	ResponseTimeMs    *int
	RequestParams     json.RawMessage
	ResponseSizeBytes *int
	ErrorMessage      *string
	IngestionRunID    *int
	Success           bool
}

func (db *DB) RecordSAMGovRequest(ctx context.Context, req *SAMGovRequest) (int, error) {
	var id int
	err := db.pool.QueryRow(ctx, `
		INSERT INTO pipeline.samgov_requests (
			endpoint,
			method,
			http_status_code,
			response_time_ms,
			request_params,
			response_size_bytes,
			error_message,
			ingestion_run_id,
			success
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
		RETURNING id
	`,
		req.Endpoint,
		req.Method,
		req.HTTPStatusCode,
		req.ResponseTimeMs,
		req.RequestParams,
		req.ResponseSizeBytes,
		req.ErrorMessage,
		req.IngestionRunID,
		req.Success,
	).Scan(&id)

	if err != nil {
		return 0, fmt.Errorf("failed to record SAM.gov request: %w", err)
	}
	return id, nil
}

func (db *DB) CountSAMGovRequestsSince(ctx context.Context, duration time.Duration) (int, error) {
	var count int
	since := time.Now().Add(-duration)
	err := db.pool.QueryRow(ctx, `
		SELECT COUNT(*) FROM pipeline.samgov_requests
		WHERE request_timestamp >= $1
	`, since).Scan(&count)

	if err != nil {
		return 0, fmt.Errorf("failed to count SAM.gov requests: %w", err)
	}
	return count, nil
}

func (db *DB) CountSuccessfulSAMGovRequestsSince(ctx context.Context, duration time.Duration) (int, error) {
	var count int
	since := time.Now().Add(-duration)
	err := db.pool.QueryRow(ctx, `
		SELECT COUNT(*) FROM pipeline.samgov_requests
		WHERE request_timestamp >= $1 AND success = true
	`, since).Scan(&count)

	if err != nil {
		return 0, fmt.Errorf("failed to count successful SAM.gov requests: %w", err)
	}
	return count, nil
}

func (db *DB) CountFailedSAMGovRequestsSince(ctx context.Context, duration time.Duration) (int, error) {
	var count int
	since := time.Now().Add(-duration)
	err := db.pool.QueryRow(ctx, `
		SELECT COUNT(*) FROM pipeline.samgov_requests
		WHERE request_timestamp >= $1 AND success = false
	`, since).Scan(&count)

	if err != nil {
		return 0, fmt.Errorf("failed to count failed SAM.gov requests: %w", err)
	}
	return count, nil
}

type SAMGovRequestStats struct {
	TotalRequests      int
	SuccessfulRequests int
	FailedRequests     int
	AvgResponseTimeMs  *float64
}

func (db *DB) GetSAMGovRequestStatsSince(ctx context.Context, duration time.Duration) (*SAMGovRequestStats, error) {
	since := time.Now().Add(-duration)
	stats := &SAMGovRequestStats{}

	err := db.pool.QueryRow(ctx, `
		SELECT
			COUNT(*) as total,
			COUNT(*) FILTER (WHERE success = true) as successful,
			COUNT(*) FILTER (WHERE success = false) as failed,
			AVG(response_time_ms) FILTER (WHERE response_time_ms IS NOT NULL) as avg_response_time
		FROM pipeline.samgov_requests
		WHERE request_timestamp >= $1
	`, since).Scan(&stats.TotalRequests, &stats.SuccessfulRequests, &stats.FailedRequests, &stats.AvgResponseTimeMs)

	if err != nil {
		return nil, fmt.Errorf("failed to get SAM.gov request stats: %w", err)
	}
	return stats, nil
}

func (db *DB) UpdateSAMGovRequestResponseSize(ctx context.Context, requestID int, sizeBytes int) error {
	_, err := db.pool.Exec(ctx, `
		UPDATE pipeline.samgov_requests
		SET response_size_bytes = $2
		WHERE id = $1
	`, requestID, sizeBytes)

	if err != nil {
		return fmt.Errorf("failed to update SAM.gov request response size: %w", err)
	}
	return nil
}
