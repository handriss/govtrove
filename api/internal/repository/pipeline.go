package repository

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
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

// --- Data Quality ---

type DataQualityRow struct {
	ID             int64   `json:"id"`
	NoticeID       string  `json:"notice_id"`
	SnapshotDate   string  `json:"snapshot_date"`
	Source         string  `json:"source"`
	IssueType      string  `json:"issue_type"`
	FieldName      *string `json:"field_name"`
	FieldValue     *string `json:"field_value"`
	Description    *string `json:"description"`
	Resolved       bool    `json:"resolved"`
	ResolvedAt     *string `json:"resolved_at"`
	ResolutionNote *string `json:"resolution_note"`
	CreatedAt      string  `json:"created_at"`
}

type ReconcileDQRow struct {
	ID             int64   `json:"id"`
	NoticeID       string  `json:"notice_id"`
	SnapshotDate   string  `json:"snapshot_date"`
	IssueType      string  `json:"issue_type"`
	FieldName      string  `json:"field_name"`
	CsvValue       *string `json:"csv_value"`
	ApiValue       *string `json:"api_value"`
	Resolved       bool    `json:"resolved"`
	ResolvedAt     *string `json:"resolved_at"`
	ResolutionNote *string `json:"resolution_note"`
	CreatedAt      string  `json:"created_at"`
}

type DataQualityDetail struct {
	DataQualityRow
	OppTitle   *string `json:"opp_title"`
	OppSolNum  *string `json:"opp_sol_num"`
	OppType    *string `json:"opp_type"`
	OppActive  *bool   `json:"opp_active"`
	OppUILink  *string `json:"opp_ui_link"`
}

type ReconcileDQDetail struct {
	ReconcileDQRow
	OppTitle   *string `json:"opp_title"`
	OppSolNum  *string `json:"opp_sol_num"`
	OppType    *string `json:"opp_type"`
	OppActive  *bool   `json:"opp_active"`
	OppUILink  *string `json:"opp_ui_link"`
}

var dqSortColumns = map[string]string{
	"date":       "dq.snapshot_date",
	"notice_id":  "dq.notice_id",
	"source":     "dq.source",
	"issue_type": "dq.issue_type",
	"field":      "dq.field_name",
	"resolved":   "dq.resolved",
	"created_at": "dq.created_at",
}

var reconcileDQSortColumns = map[string]string{
	"date":       "dq.snapshot_date",
	"notice_id":  "dq.notice_id",
	"issue_type": "dq.issue_type",
	"field":      "dq.field_name",
	"resolved":   "dq.resolved",
	"created_at": "dq.created_at",
}

func (r *PipelineRepository) ListDataQualityIssues(ctx context.Context, page, limit int, sort, order string, resolved *bool) ([]DataQualityRow, int, error) {
	where := ""
	var args []any
	if resolved != nil {
		where = "WHERE dq.resolved = $1"
		args = append(args, *resolved)
	}

	var total int
	countQ := "SELECT COUNT(*) FROM pipeline.snap_data_quality dq " + where
	if err := r.pool.QueryRow(ctx, countQ, args...).Scan(&total); err != nil {
		return nil, 0, err
	}

	col, ok := dqSortColumns[sort]
	if !ok {
		col = "dq.snapshot_date"
	}
	dir := "DESC"
	if order == "asc" {
		dir = "ASC"
	}

	offset := (page - 1) * limit
	nextParam := len(args) + 1
	query := fmt.Sprintf(`
		SELECT dq.id, dq.notice_id, dq.snapshot_date, dq.source, dq.issue_type,
		       dq.field_name, dq.field_value, dq.description, dq.resolved,
		       dq.resolved_at, dq.resolution_note, dq.created_at
		FROM pipeline.snap_data_quality dq
		%s
		ORDER BY %s %s
		LIMIT $%d OFFSET $%d
	`, where, col, dir, nextParam, nextParam+1)
	args = append(args, limit, offset)

	rows, err := r.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var items []DataQualityRow
	for rows.Next() {
		var row DataQualityRow
		var snapDate, createdAt time.Time
		var resolvedAt *time.Time
		if err := rows.Scan(&row.ID, &row.NoticeID, &snapDate, &row.Source, &row.IssueType,
			&row.FieldName, &row.FieldValue, &row.Description, &row.Resolved,
			&resolvedAt, &row.ResolutionNote, &createdAt); err != nil {
			return nil, 0, err
		}
		row.SnapshotDate = snapDate.Format(time.RFC3339)
		row.CreatedAt = createdAt.Format(time.RFC3339)
		if resolvedAt != nil {
			s := resolvedAt.Format(time.RFC3339)
			row.ResolvedAt = &s
		}
		items = append(items, row)
	}
	return items, total, rows.Err()
}

func (r *PipelineRepository) ListReconcileDQIssues(ctx context.Context, page, limit int, sort, order string, resolved *bool) ([]ReconcileDQRow, int, error) {
	where := ""
	var args []any
	if resolved != nil {
		where = "WHERE dq.resolved = $1"
		args = append(args, *resolved)
	}

	var total int
	countQ := "SELECT COUNT(*) FROM pipeline.snap_reconcile_dq dq " + where
	if err := r.pool.QueryRow(ctx, countQ, args...).Scan(&total); err != nil {
		return nil, 0, err
	}

	col, ok := reconcileDQSortColumns[sort]
	if !ok {
		col = "dq.snapshot_date"
	}
	dir := "DESC"
	if order == "asc" {
		dir = "ASC"
	}

	offset := (page - 1) * limit
	nextParam := len(args) + 1
	query := fmt.Sprintf(`
		SELECT dq.id, dq.notice_id, dq.snapshot_date, dq.issue_type, dq.field_name,
		       dq.csv_value, dq.api_value, dq.resolved,
		       dq.resolved_at, dq.resolution_note, dq.created_at
		FROM pipeline.snap_reconcile_dq dq
		%s
		ORDER BY %s %s
		LIMIT $%d OFFSET $%d
	`, where, col, dir, nextParam, nextParam+1)
	args = append(args, limit, offset)

	rows, err := r.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var items []ReconcileDQRow
	for rows.Next() {
		var row ReconcileDQRow
		var snapDate, createdAt time.Time
		var resolvedAt *time.Time
		if err := rows.Scan(&row.ID, &row.NoticeID, &snapDate, &row.IssueType, &row.FieldName,
			&row.CsvValue, &row.ApiValue, &row.Resolved,
			&resolvedAt, &row.ResolutionNote, &createdAt); err != nil {
			return nil, 0, err
		}
		row.SnapshotDate = snapDate.Format(time.RFC3339)
		row.CreatedAt = createdAt.Format(time.RFC3339)
		if resolvedAt != nil {
			s := resolvedAt.Format(time.RFC3339)
			row.ResolvedAt = &s
		}
		items = append(items, row)
	}
	return items, total, rows.Err()
}

func (r *PipelineRepository) GetDataQualityDetail(ctx context.Context, id int64) (*DataQualityDetail, error) {
	var d DataQualityDetail
	var snapDate, createdAt time.Time
	var resolvedAt *time.Time
	err := r.pool.QueryRow(ctx, `
		SELECT dq.id, dq.notice_id, dq.snapshot_date, dq.source, dq.issue_type,
		       dq.field_name, dq.field_value, dq.description, dq.resolved,
		       dq.resolved_at, dq.resolution_note, dq.created_at,
		       o.title, o.solicitation_number, o.type, o.active, o.ui_link
		FROM pipeline.snap_data_quality dq
		LEFT JOIN opportunities o ON o.notice_id = dq.notice_id AND o.is_latest = true
		WHERE dq.id = $1
	`, id).Scan(
		&d.ID, &d.NoticeID, &snapDate, &d.Source, &d.IssueType,
		&d.FieldName, &d.FieldValue, &d.Description, &d.Resolved,
		&resolvedAt, &d.ResolutionNote, &createdAt,
		&d.OppTitle, &d.OppSolNum, &d.OppType, &d.OppActive, &d.OppUILink,
	)
	if err != nil {
		return nil, err
	}
	d.SnapshotDate = snapDate.Format(time.RFC3339)
	d.CreatedAt = createdAt.Format(time.RFC3339)
	if resolvedAt != nil {
		s := resolvedAt.Format(time.RFC3339)
		d.ResolvedAt = &s
	}
	return &d, nil
}

func (r *PipelineRepository) GetReconcileDQDetail(ctx context.Context, id int64) (*ReconcileDQDetail, error) {
	var d ReconcileDQDetail
	var snapDate, createdAt time.Time
	var resolvedAt *time.Time
	err := r.pool.QueryRow(ctx, `
		SELECT dq.id, dq.notice_id, dq.snapshot_date, dq.issue_type, dq.field_name,
		       dq.csv_value, dq.api_value, dq.resolved,
		       dq.resolved_at, dq.resolution_note, dq.created_at,
		       o.title, o.solicitation_number, o.type, o.active, o.ui_link
		FROM pipeline.snap_reconcile_dq dq
		LEFT JOIN opportunities o ON o.notice_id = dq.notice_id AND o.is_latest = true
		WHERE dq.id = $1
	`, id).Scan(
		&d.ID, &d.NoticeID, &snapDate, &d.IssueType, &d.FieldName,
		&d.CsvValue, &d.ApiValue, &d.Resolved,
		&resolvedAt, &d.ResolutionNote, &createdAt,
		&d.OppTitle, &d.OppSolNum, &d.OppType, &d.OppActive, &d.OppUILink,
	)
	if err != nil {
		return nil, err
	}
	d.SnapshotDate = snapDate.Format(time.RFC3339)
	d.CreatedAt = createdAt.Format(time.RFC3339)
	if resolvedAt != nil {
		s := resolvedAt.Format(time.RFC3339)
		d.ResolvedAt = &s
	}
	return &d, nil
}

func (r *PipelineRepository) UpdateDataQualityResolution(ctx context.Context, id int64, resolved bool, note string) error {
	var resolvedAt *time.Time
	if resolved {
		now := time.Now()
		resolvedAt = &now
	}
	_, err := r.pool.Exec(ctx, `
		UPDATE pipeline.snap_data_quality
		SET resolved = $2, resolved_at = $3, resolution_note = $4
		WHERE id = $1
	`, id, resolved, resolvedAt, note)
	return err
}

func (r *PipelineRepository) UpdateReconcileDQResolution(ctx context.Context, id int64, resolved bool, note string) error {
	var resolvedAt *time.Time
	if resolved {
		now := time.Now()
		resolvedAt = &now
	}
	_, err := r.pool.Exec(ctx, `
		UPDATE pipeline.snap_reconcile_dq
		SET resolved = $2, resolved_at = $3, resolution_note = $4
		WHERE id = $1
	`, id, resolved, resolvedAt, note)
	return err
}

// --- Pipeline Run Detail ---

type IngestionRunDetail struct {
	RunID           string  `json:"run_id"`
	JobType         string  `json:"job_type"`
	Status          string  `json:"status"`
	StartedAt       string  `json:"started_at"`
	CompletedAt     *string `json:"completed_at"`
	RecordsFetched  *int    `json:"records_fetched"`
	RecordsInserted *int    `json:"records_inserted"`
	RecordsUpdated  *int    `json:"records_updated"`
	RecordsFailed   *int    `json:"records_failed"`
	RecordsSkipped  *int    `json:"records_skipped"`
	DurationMs      *int    `json:"duration_ms"`
	ErrorMessage    *string `json:"error_message"`
}

type TableCounts struct {
	SnapCSV          int `json:"snap_csv"`
	SnapAPI          int `json:"snap_api"`
	SnapDataQuality  int `json:"snap_data_quality"`
	Disappearances   int `json:"snap_disappearances"`
	ReconcileDQ      int `json:"snap_reconcile_dq"`
}

type OpportunityStats struct {
	TotalAffected int `json:"total_affected"`
	Inserted      int `json:"inserted"`
	Updated       int `json:"updated"`
	FromCSVOnly   int `json:"from_csv_only"`
	FromAPI       int `json:"from_api"`
	FromBoth      int `json:"from_both"`
}

type PipelineRunDetailResponse struct {
	PipelineRun      PipelineRunRow     `json:"pipeline_run"`
	IngestionRuns    []IngestionRunDetail `json:"ingestion_runs"`
	TableCounts      TableCounts        `json:"table_counts"`
	OpportunityStats OpportunityStats   `json:"opportunity_stats"`
}

var ErrPipelineRunNotFound = errors.New("pipeline run not found")

func (r *PipelineRepository) GetPipelineRunDetail(ctx context.Context, id string) (*PipelineRunDetailResponse, error) {
	// Q1: pipeline run by ID
	var run PipelineRunRow
	var startedAt time.Time
	var completedAt *time.Time
	err := r.pool.QueryRow(ctx, `
		SELECT id, pipeline_name, status, started_at, completed_at, duration_ms, error_message
		FROM pipeline.pipeline_runs WHERE id = $1
	`, id).Scan(&run.ID, &run.PipelineName, &run.Status, &startedAt, &completedAt,
		&run.DurationMs, &run.ErrorMessage)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrPipelineRunNotFound
		}
		return nil, err
	}
	run.StartedAt = startedAt.Format(time.RFC3339)
	if completedAt != nil {
		s := completedAt.Format(time.RFC3339)
		run.CompletedAt = &s
	}

	// Build reusable time-window args: $1=startedAt, optionally $2=completedAt
	endExpr := "NOW()"
	windowArgs := []any{startedAt}
	if completedAt != nil {
		endExpr = "$2"
		windowArgs = append(windowArgs, *completedAt)
	}

	// Q2: ingestion runs in the time window
	irQuery := fmt.Sprintf(`
		SELECT run_id, job_type, status, started_at, completed_at,
		       records_fetched, records_inserted, records_updated,
		       records_failed, records_skipped, duration_ms, error_message
		FROM pipeline.ingestion_runs
		WHERE started_at BETWEEN $1 AND %s
		ORDER BY started_at ASC
	`, endExpr)
	irRows, err := r.pool.Query(ctx, irQuery, windowArgs...)
	if err != nil {
		return nil, err
	}
	defer irRows.Close()

	var ingestionRuns []IngestionRunDetail
	for irRows.Next() {
		var ir IngestionRunDetail
		var sa time.Time
		var ca *time.Time
		if err := irRows.Scan(&ir.RunID, &ir.JobType, &ir.Status, &sa, &ca,
			&ir.RecordsFetched, &ir.RecordsInserted, &ir.RecordsUpdated,
			&ir.RecordsFailed, &ir.RecordsSkipped, &ir.DurationMs, &ir.ErrorMessage); err != nil {
			return nil, err
		}
		ir.StartedAt = sa.Format(time.RFC3339)
		if ca != nil {
			s := ca.Format(time.RFC3339)
			ir.CompletedAt = &s
		}
		ingestionRuns = append(ingestionRuns, ir)
	}
	if err := irRows.Err(); err != nil {
		return nil, err
	}

	// Q3: table counts
	tcQuery := fmt.Sprintf(`
		WITH run_ids AS (
			SELECT ir.run_id FROM pipeline.ingestion_runs ir
			WHERE ir.started_at BETWEEN $1 AND %s
		)
		SELECT
			(SELECT COUNT(*) FROM pipeline.snap_csv WHERE run_id IN (SELECT run_id FROM run_ids)),
			(SELECT COUNT(*) FROM pipeline.snap_api WHERE run_id IN (SELECT run_id FROM run_ids)),
			(SELECT COUNT(*) FROM pipeline.snap_data_quality WHERE run_id IN (SELECT run_id FROM run_ids)),
			(SELECT COUNT(*) FROM pipeline.snap_disappearances WHERE run_id IN (SELECT run_id FROM run_ids)),
			(SELECT COUNT(*) FROM pipeline.snap_reconcile_dq WHERE csv_run_id IN (SELECT run_id FROM run_ids) OR api_run_id IN (SELECT run_id FROM run_ids))
	`, endExpr)
	var tc TableCounts
	if err := r.pool.QueryRow(ctx, tcQuery, windowArgs...).Scan(
		&tc.SnapCSV, &tc.SnapAPI, &tc.SnapDataQuality, &tc.Disappearances, &tc.ReconcileDQ,
	); err != nil {
		return nil, err
	}

	// Q4: opportunity stats
	osQuery := fmt.Sprintf(`
		WITH run_ids AS (
			SELECT ir.run_id FROM pipeline.ingestion_runs ir
			WHERE ir.started_at BETWEEN $1 AND %s
		)
		SELECT
			COUNT(*),
			COUNT(*) FILTER (WHERE created_at >= $1),
			COUNT(*) FILTER (WHERE created_at < $1)
		FROM opportunities
		WHERE last_csv_run_id IN (SELECT run_id FROM run_ids)
	`, endExpr)
	var oppStats OpportunityStats
	if err := r.pool.QueryRow(ctx, osQuery, windowArgs...).Scan(
		&oppStats.TotalAffected, &oppStats.Inserted, &oppStats.Updated,
	); err != nil {
		return nil, err
	}

	// from_both: notice_ids appearing in both snap_csv and snap_api
	fbQuery := fmt.Sprintf(`
		WITH run_ids AS (
			SELECT ir.run_id FROM pipeline.ingestion_runs ir
			WHERE ir.started_at BETWEEN $1 AND %s
		)
		SELECT COUNT(DISTINCT sa.notice_id)
		FROM pipeline.snap_api sa
		WHERE sa.run_id IN (SELECT run_id FROM run_ids)
		  AND sa.notice_id IN (SELECT notice_id FROM pipeline.snap_csv WHERE run_id IN (SELECT run_id FROM run_ids))
	`, endExpr)
	if err := r.pool.QueryRow(ctx, fbQuery, windowArgs...).Scan(&oppStats.FromBoth); err != nil {
		return nil, err
	}
	oppStats.FromAPI = tc.SnapAPI
	oppStats.FromCSVOnly = oppStats.TotalAffected - oppStats.FromBoth

	return &PipelineRunDetailResponse{
		PipelineRun:      run,
		IngestionRuns:    ingestionRuns,
		TableCounts:      tc,
		OpportunityStats: oppStats,
	}, nil
}
