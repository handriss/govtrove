package repository

import (
	"context"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

type PipelineRepository struct {
	pool *pgxpool.Pool
}

func NewPipelineRepository(pool *pgxpool.Pool) *PipelineRepository {
	return &PipelineRepository{pool: pool}
}

type ApiKeyRow struct {
	KeyHash    string    `json:"key_hash"`
	Email      string    `json:"email"`
	DailyLimit int       `json:"daily_limit"`
	CreatedAt  time.Time `json:"created_at"`
	ExpiresAt  time.Time `json:"expires_at"`
}

type SamgovRequestRow struct {
	ID               int     `json:"id"`
	RequestTimestamp string  `json:"request_timestamp"`
	Endpoint         string  `json:"endpoint"`
	Method           string  `json:"method"`
	HTTPStatusCode   *int    `json:"http_status_code"`
	ResponseTimeMs   *int    `json:"response_time_ms"`
	RequestParams    *string `json:"request_params"`
	ResponseSize     *int    `json:"response_size_bytes"`
	ErrorMessage     *string `json:"error_message"`
	Success          bool    `json:"success"`
	ApiKeyHash       *string `json:"api_key_hash"`
}

type UsageBucket struct {
	Timestamp time.Time `json:"timestamp"`
	Success   int       `json:"success"`
	Failed    int       `json:"failed"`
}

func (r *PipelineRepository) ListApiKeys(ctx context.Context) ([]ApiKeyRow, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT key_hash, email, daily_limit, created_at, expires_at
		FROM pipeline.api_keys
		ORDER BY created_at DESC
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var keys []ApiKeyRow
	for rows.Next() {
		var k ApiKeyRow
		if err := rows.Scan(&k.KeyHash, &k.Email, &k.DailyLimit, &k.CreatedAt, &k.ExpiresAt); err != nil {
			return nil, err
		}
		keys = append(keys, k)
	}
	return keys, rows.Err()
}

func (r *PipelineRepository) ListSamgovRequests(ctx context.Context, page, limit int) ([]SamgovRequestRow, int, error) {
	var total int
	err := r.pool.QueryRow(ctx, `SELECT COUNT(*) FROM pipeline.samgov_requests`).Scan(&total)
	if err != nil {
		return nil, 0, err
	}

	offset := (page - 1) * limit
	rows, err := r.pool.Query(ctx, `
		SELECT id, request_timestamp, endpoint, method, http_status_code,
		       response_time_ms, request_params::text, response_size_bytes,
		       error_message, success, api_key_hash
		FROM pipeline.samgov_requests
		ORDER BY request_timestamp DESC
		LIMIT $1 OFFSET $2
	`, limit, offset)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var reqs []SamgovRequestRow
	for rows.Next() {
		var r SamgovRequestRow
		var ts time.Time
		if err := rows.Scan(&r.ID, &ts, &r.Endpoint, &r.Method, &r.HTTPStatusCode,
			&r.ResponseTimeMs, &r.RequestParams, &r.ResponseSize,
			&r.ErrorMessage, &r.Success, &r.ApiKeyHash); err != nil {
			return nil, 0, err
		}
		r.RequestTimestamp = ts.Format(time.RFC3339)
		reqs = append(reqs, r)
	}
	return reqs, total, rows.Err()
}

type PipelineRunRow struct {
	ID             string  `json:"id"`
	PipelineName   string  `json:"pipeline_name"`
	Status         string  `json:"status"`
	StartedAt      string  `json:"started_at"`
	CompletedAt    *string `json:"completed_at"`
	DurationMs     *int    `json:"duration_ms"`
	ErrorMessage   *string `json:"error_message"`
	IngestionRuns  *string `json:"ingestion_runs"`
}

type SearchEventRow struct {
	ID           int     `json:"id"`
	Query        *string `json:"query"`
	Filters      *string `json:"filters"`
	SortBy       *string `json:"sort_by"`
	Page         *int    `json:"page"`
	TotalResults *int    `json:"total_results"`
	UserID       *string `json:"user_id"`
	CreatedAt    string  `json:"created_at"`
}

func (r *PipelineRepository) ListPipelineRuns(ctx context.Context, page, limit int, fullOnly bool) ([]PipelineRunRow, int, error) {
	countQuery := `SELECT COUNT(*) FROM pipeline.pipeline_runs`
	if fullOnly {
		countQuery += ` WHERE EXISTS (
			SELECT 1 FROM pipeline.ingestion_runs ir
			WHERE ir.started_at BETWEEN pipeline_runs.started_at AND COALESCE(pipeline_runs.completed_at, NOW())
		)`
	}

	var total int
	if err := r.pool.QueryRow(ctx, countQuery).Scan(&total); err != nil {
		return nil, 0, err
	}

	fullOnlyClause := ""
	if fullOnly {
		fullOnlyClause = `WHERE EXISTS (
			SELECT 1 FROM pipeline.ingestion_runs ir2
			WHERE ir2.started_at BETWEEN pr.started_at AND COALESCE(pr.completed_at, NOW())
		)`
	}

	offset := (page - 1) * limit
	query := `
		SELECT pr.id, pr.pipeline_name, pr.status, pr.started_at, pr.completed_at,
		       pr.duration_ms, pr.error_message,
		       (SELECT json_agg(json_build_object(
		           'run_id', ir.run_id,
		           'job_type', ir.job_type,
		           'status', ir.status,
		           'records_fetched', ir.records_fetched,
		           'records_inserted', ir.records_inserted,
		           'records_updated', ir.records_updated,
		           'records_failed', ir.records_failed,
		           'records_skipped', ir.records_skipped,
		           'duration_ms', ir.duration_ms,
		           'error_message', ir.error_message
		       )) FILTER (WHERE ir.run_id IS NOT NULL)
		       FROM pipeline.ingestion_runs ir
		       WHERE ir.started_at BETWEEN pr.started_at AND COALESCE(pr.completed_at, NOW())
		       )::text AS ingestion_runs
		FROM pipeline.pipeline_runs pr
		` + fullOnlyClause + `
		ORDER BY pr.started_at DESC
		LIMIT $1 OFFSET $2
	`

	rows, err := r.pool.Query(ctx, query, limit, offset)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var runs []PipelineRunRow
	for rows.Next() {
		var run PipelineRunRow
		var startedAt time.Time
		var completedAt *time.Time
		if err := rows.Scan(&run.ID, &run.PipelineName, &run.Status, &startedAt, &completedAt,
			&run.DurationMs, &run.ErrorMessage, &run.IngestionRuns); err != nil {
			return nil, 0, err
		}
		run.StartedAt = startedAt.Format(time.RFC3339)
		if completedAt != nil {
			s := completedAt.Format(time.RFC3339)
			run.CompletedAt = &s
		}
		runs = append(runs, run)
	}
	return runs, total, rows.Err()
}

func (r *PipelineRepository) ListSearchEvents(ctx context.Context, page, limit int, emptyOnly bool) ([]SearchEventRow, int, error) {
	countQuery := `SELECT COUNT(*) FROM search_events WHERE event_type = 'search'`
	if emptyOnly {
		countQuery += ` AND (total_results = 0 OR total_results IS NULL)`
	}

	var total int
	if err := r.pool.QueryRow(ctx, countQuery).Scan(&total); err != nil {
		return nil, 0, err
	}

	whereClause := `WHERE event_type = 'search'`
	if emptyOnly {
		whereClause += ` AND (total_results = 0 OR total_results IS NULL)`
	}

	offset := (page - 1) * limit
	query := `
		SELECT id, query, filters::text, sort_by, page, total_results, user_id, created_at
		FROM search_events
		` + whereClause + `
		ORDER BY created_at DESC
		LIMIT $1 OFFSET $2
	`

	rows, err := r.pool.Query(ctx, query, limit, offset)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var events []SearchEventRow
	for rows.Next() {
		var ev SearchEventRow
		var createdAt time.Time
		if err := rows.Scan(&ev.ID, &ev.Query, &ev.Filters, &ev.SortBy, &ev.Page,
			&ev.TotalResults, &ev.UserID, &createdAt); err != nil {
			return nil, 0, err
		}
		ev.CreatedAt = createdAt.Format(time.RFC3339)
		events = append(events, ev)
	}
	return events, total, rows.Err()
}

func (r *PipelineRepository) GetApiKeyUsage(ctx context.Context, keyHash string, days int) ([]UsageBucket, error) {
	rows, err := r.pool.Query(ctx, `
		WITH buckets AS (
			SELECT generate_series(
				date_trunc('hour', NOW()) - ($2::int * INTERVAL '1 day') + INTERVAL '1 hour',
				date_trunc('hour', NOW()),
				INTERVAL '1 hour'
			) AS bucket_time
		)
		SELECT b.bucket_time,
			COUNT(r.id) FILTER (WHERE r.success = true),
			COUNT(r.id) FILTER (WHERE r.success = false)
		FROM buckets b
		LEFT JOIN pipeline.samgov_requests r
			ON r.api_key_hash = $1
			AND r.request_timestamp > b.bucket_time - INTERVAL '24 hours'
			AND r.request_timestamp <= b.bucket_time
		GROUP BY b.bucket_time
		ORDER BY b.bucket_time
	`, keyHash, days)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var buckets []UsageBucket
	for rows.Next() {
		var b UsageBucket
		if err := rows.Scan(&b.Timestamp, &b.Success, &b.Failed); err != nil {
			return nil, err
		}
		buckets = append(buckets, b)
	}
	return buckets, rows.Err()
}
