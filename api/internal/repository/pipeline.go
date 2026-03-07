package repository

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
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

type PipelineStepRow struct {
	ID           string  `json:"id"`
	StepName     string  `json:"step_name"`
	Status       string  `json:"status"`
	StartedAt    string  `json:"started_at"`
	CompletedAt  *string `json:"completed_at"`
	DurationMs   *int    `json:"duration_ms"`
	Stats        json.RawMessage `json:"stats"`
	ErrorMessage *string `json:"error_message"`
	Attempt      int     `json:"attempt"`
	IsLatest     bool    `json:"is_latest"`
}

type PipelineExecutionRow struct {
	ExecutionID string            `json:"execution_id"`
	Status      string            `json:"status"`
	StartedAt   string            `json:"started_at"`
	CompletedAt *string           `json:"completed_at"`
	DurationMs  *int              `json:"duration_ms"`
	StepCount   int               `json:"step_count"`
	Steps       []PipelineStepRow `json:"steps"`
}

type McpUsageRow struct {
	ID            int     `json:"id"`
	UserEmail     *string `json:"user_email"`
	ToolName      string  `json:"tool_name"`
	RequestParams *string `json:"request_params"`
	ResultCount   *int    `json:"result_count"`
	LatencyMs     *int    `json:"latency_ms"`
	CalledAt      string  `json:"called_at"`
}

type SearchEventRow struct {
	ID           int     `json:"id"`
	Query        *string `json:"query"`
	Filters      *string `json:"filters"`
	SortBy       *string `json:"sort_by"`
	Page         *int    `json:"page"`
	TotalResults *int    `json:"total_results"`
	UserID       *string `json:"user_id"`
	DurationMs   *int    `json:"duration_ms"`
	CreatedAt    string  `json:"created_at"`
}

func (r *PipelineRepository) ListPipelineRuns(ctx context.Context, page, limit int, _ bool) ([]PipelineExecutionRow, int, error) {
	var total int
	if err := r.pool.QueryRow(ctx, `
		SELECT COUNT(DISTINCT execution_id) FROM pipeline.pipeline_steps
	`).Scan(&total); err != nil {
		return nil, 0, err
	}

	offset := (page - 1) * limit
	rows, err := r.pool.Query(ctx, `
		WITH latest_per_step AS (
			SELECT DISTINCT ON (execution_id, step_name) execution_id, status
			FROM pipeline.pipeline_steps
			ORDER BY execution_id, step_name, started_at DESC
		)
		SELECT
			ps.execution_id,
			MIN(ps.started_at) AS started_at,
			MAX(ps.completed_at) AS completed_at,
			EXTRACT(EPOCH FROM (MAX(ps.completed_at) - MIN(ps.started_at)))::int * 1000 AS duration_ms,
			(SELECT BOOL_AND(lps.status = 'completed') FROM latest_per_step lps WHERE lps.execution_id = ps.execution_id) AS all_completed,
			(SELECT BOOL_OR(lps.status = 'failed') FROM latest_per_step lps WHERE lps.execution_id = ps.execution_id) AS any_failed,
			(SELECT BOOL_OR(lps.status = 'running') FROM latest_per_step lps WHERE lps.execution_id = ps.execution_id) AS any_running,
			COUNT(*) AS step_count,
			(SELECT json_agg(json_build_object(
				'id', ps2.id, 'step_name', ps2.step_name, 'status', ps2.status,
				'started_at', ps2.started_at, 'completed_at', ps2.completed_at,
				'duration_ms', ps2.duration_ms, 'stats', ps2.stats,
				'error_message', ps2.error_message,
				'attempt', ps2.attempt, 'is_latest', ps2.is_latest
			) ORDER BY ps2.started_at)
			FROM (
				SELECT *,
					ROW_NUMBER() OVER (PARTITION BY step_name ORDER BY started_at) AS attempt,
					(ROW_NUMBER() OVER (PARTITION BY step_name ORDER BY started_at DESC) = 1) AS is_latest
				FROM pipeline.pipeline_steps ps3
				WHERE ps3.execution_id = ps.execution_id
			) ps2
			)::text AS steps
		FROM pipeline.pipeline_steps ps
		GROUP BY ps.execution_id
		ORDER BY MIN(ps.started_at) DESC
		LIMIT $1 OFFSET $2
	`, limit, offset)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var executions []PipelineExecutionRow
	for rows.Next() {
		var exec PipelineExecutionRow
		var startedAt time.Time
		var completedAt *time.Time
		var allCompleted, anyFailed, anyRunning bool
		var stepsJSON *string
		if err := rows.Scan(&exec.ExecutionID, &startedAt, &completedAt,
			&exec.DurationMs, &allCompleted, &anyFailed, &anyRunning,
			&exec.StepCount, &stepsJSON); err != nil {
			return nil, 0, err
		}
		exec.StartedAt = startedAt.Format(time.RFC3339)
		if completedAt != nil {
			s := completedAt.Format(time.RFC3339)
			exec.CompletedAt = &s
		}
		if anyFailed {
			exec.Status = "failed"
		} else if anyRunning {
			exec.Status = "running"
		} else if allCompleted {
			exec.Status = "completed"
		} else {
			exec.Status = "running"
		}

		if stepsJSON != nil {
			var steps []PipelineStepRow
			if err := json.Unmarshal([]byte(*stepsJSON), &steps); err == nil {
				for i := range steps {
					if steps[i].StartedAt != "" {
						if t, err := time.Parse(time.RFC3339Nano, steps[i].StartedAt); err == nil {
							steps[i].StartedAt = t.Format(time.RFC3339)
						}
					}
					if steps[i].CompletedAt != nil {
						if t, err := time.Parse(time.RFC3339Nano, *steps[i].CompletedAt); err == nil {
							s := t.Format(time.RFC3339)
							steps[i].CompletedAt = &s
						}
					}
				}
				exec.Steps = steps
			}
		}
		if exec.Steps == nil {
			exec.Steps = []PipelineStepRow{}
		}

		executions = append(executions, exec)
	}
	return executions, total, rows.Err()
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
		SELECT id, query, filters::text, sort_by, page, total_results, user_id, duration_ms, created_at
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
			&ev.TotalResults, &ev.UserID, &ev.DurationMs, &createdAt); err != nil {
			return nil, 0, err
		}
		ev.CreatedAt = createdAt.Format(time.RFC3339)
		events = append(events, ev)
	}
	return events, total, rows.Err()
}

func (r *PipelineRepository) ListMcpUsage(ctx context.Context, page, limit int) ([]McpUsageRow, int, error) {
	var total int
	if err := r.pool.QueryRow(ctx, `SELECT COUNT(*) FROM mcp_usage`).Scan(&total); err != nil {
		return nil, 0, err
	}

	offset := (page - 1) * limit
	rows, err := r.pool.Query(ctx, `
		SELECT id, user_email, tool_name, request_params::text, result_count, latency_ms, called_at
		FROM mcp_usage
		ORDER BY called_at DESC
		LIMIT $1 OFFSET $2
	`, limit, offset)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var events []McpUsageRow
	for rows.Next() {
		var ev McpUsageRow
		var calledAt time.Time
		if err := rows.Scan(&ev.ID, &ev.UserEmail, &ev.ToolName, &ev.RequestParams,
			&ev.ResultCount, &ev.LatencyMs, &calledAt); err != nil {
			return nil, 0, err
		}
		ev.CalledAt = calledAt.Format(time.RFC3339)
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

func (r *PipelineRepository) ListDataQualityIssues(ctx context.Context, page, limit int, sort, order string, resolved *bool, snapshotDate, issueType, fieldName string) ([]DataQualityRow, int, error) {
	var conditions []string
	var args []any
	if resolved != nil {
		args = append(args, *resolved)
		conditions = append(conditions, fmt.Sprintf("dq.resolved = $%d", len(args)))
	}
	if snapshotDate != "" {
		args = append(args, snapshotDate)
		conditions = append(conditions, fmt.Sprintf("dq.snapshot_date::date = $%d::date", len(args)))
	}
	if issueType != "" {
		args = append(args, issueType)
		conditions = append(conditions, fmt.Sprintf("dq.issue_type = $%d", len(args)))
	}
	if fieldName != "" {
		args = append(args, fieldName)
		conditions = append(conditions, fmt.Sprintf("dq.field_name = $%d", len(args)))
	}
	where := ""
	if len(conditions) > 0 {
		where = "WHERE " + strings.Join(conditions, " AND ")
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

func (r *PipelineRepository) ListReconcileDQIssues(ctx context.Context, page, limit int, sort, order string, resolved *bool, snapshotDate, issueType, fieldName string) ([]ReconcileDQRow, int, error) {
	var conditions []string
	var args []any
	if resolved != nil {
		args = append(args, *resolved)
		conditions = append(conditions, fmt.Sprintf("dq.resolved = $%d", len(args)))
	}
	if snapshotDate != "" {
		args = append(args, snapshotDate)
		conditions = append(conditions, fmt.Sprintf("dq.snapshot_date::date = $%d::date", len(args)))
	}
	if issueType != "" {
		args = append(args, issueType)
		conditions = append(conditions, fmt.Sprintf("dq.issue_type = $%d", len(args)))
	}
	if fieldName != "" {
		args = append(args, fieldName)
		conditions = append(conditions, fmt.Sprintf("dq.field_name = $%d", len(args)))
	}
	where := ""
	if len(conditions) > 0 {
		where = "WHERE " + strings.Join(conditions, " AND ")
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

type DQSummaryRow struct {
	SnapshotDate string  `json:"snapshot_date"`
	IssueType    string  `json:"issue_type"`
	FieldName    *string `json:"field_name"`
	Total        int     `json:"total"`
	Unresolved   int     `json:"unresolved"`
}

func (r *PipelineRepository) DataQualitySummary(ctx context.Context) ([]DQSummaryRow, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT snapshot_date::date, issue_type, field_name,
		       COUNT(*) AS total,
		       COUNT(*) FILTER (WHERE NOT resolved) AS unresolved
		FROM pipeline.snap_data_quality
		GROUP BY snapshot_date::date, issue_type, field_name
		ORDER BY snapshot_date::date DESC, issue_type, field_name
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var items []DQSummaryRow
	for rows.Next() {
		var row DQSummaryRow
		var d time.Time
		if err := rows.Scan(&d, &row.IssueType, &row.FieldName, &row.Total, &row.Unresolved); err != nil {
			return nil, err
		}
		row.SnapshotDate = d.Format("2006-01-02")
		items = append(items, row)
	}
	return items, rows.Err()
}

func (r *PipelineRepository) ReconcileDQSummary(ctx context.Context) ([]DQSummaryRow, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT snapshot_date::date, issue_type, field_name,
		       COUNT(*) AS total,
		       COUNT(*) FILTER (WHERE NOT resolved) AS unresolved
		FROM pipeline.snap_reconcile_dq
		GROUP BY snapshot_date::date, issue_type, field_name
		ORDER BY snapshot_date::date DESC, issue_type, field_name
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var items []DQSummaryRow
	for rows.Next() {
		var row DQSummaryRow
		var d time.Time
		if err := rows.Scan(&d, &row.IssueType, &row.FieldName, &row.Total, &row.Unresolved); err != nil {
			return nil, err
		}
		row.SnapshotDate = d.Format("2006-01-02")
		items = append(items, row)
	}
	return items, rows.Err()
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
	ExecutionID      string             `json:"execution_id"`
	Status           string             `json:"status"`
	StartedAt        string             `json:"started_at"`
	CompletedAt      *string            `json:"completed_at"`
	DurationMs       *int               `json:"duration_ms"`
	Steps            []PipelineStepRow  `json:"steps"`
	TableCounts      TableCounts        `json:"table_counts"`
	OpportunityStats OpportunityStats   `json:"opportunity_stats"`
}

var ErrPipelineRunNotFound = errors.New("pipeline run not found")

func (r *PipelineRepository) GetPipelineRunDetail(ctx context.Context, executionID string) (*PipelineRunDetailResponse, error) {
	// Q1: pipeline steps for this execution (with attempt number and is_latest flag)
	stepRows, err := r.pool.Query(ctx, `
		SELECT id, step_name, status, started_at, completed_at, duration_ms, stats, error_message,
			ROW_NUMBER() OVER (PARTITION BY step_name ORDER BY started_at) AS attempt,
			(ROW_NUMBER() OVER (PARTITION BY step_name ORDER BY started_at DESC) = 1) AS is_latest
		FROM pipeline.pipeline_steps
		WHERE execution_id = $1
		ORDER BY started_at ASC
	`, executionID)
	if err != nil {
		return nil, err
	}
	defer stepRows.Close()

	var steps []PipelineStepRow
	var firstStartedAt time.Time
	var lastCompletedAt *time.Time

	for stepRows.Next() {
		var s PipelineStepRow
		var sa time.Time
		var ca *time.Time
		if err := stepRows.Scan(&s.ID, &s.StepName, &s.Status, &sa, &ca,
			&s.DurationMs, &s.Stats, &s.ErrorMessage, &s.Attempt, &s.IsLatest); err != nil {
			return nil, err
		}
		s.StartedAt = sa.Format(time.RFC3339)
		if ca != nil {
			cs := ca.Format(time.RFC3339)
			s.CompletedAt = &cs
		}

		if len(steps) == 0 {
			firstStartedAt = sa
		}
		if ca != nil && (lastCompletedAt == nil || ca.After(*lastCompletedAt)) {
			lastCompletedAt = ca
		}

		steps = append(steps, s)
	}
	if err := stepRows.Err(); err != nil {
		return nil, err
	}
	if len(steps) == 0 {
		return nil, ErrPipelineRunNotFound
	}

	// Derive overall status from the latest attempt per step_name.
	// A step stuck as "running" when a later step already started means
	// it completed but failed to write its status (e.g. Lambda context expired).
	type stepInfo struct {
		Status    string
		StartedAt time.Time
	}
	latestByName := make(map[string]stepInfo)
	for _, s := range steps {
		if s.IsLatest {
			sa, _ := time.Parse(time.RFC3339, s.StartedAt)
			latestByName[s.StepName] = stepInfo{Status: s.Status, StartedAt: sa}
		}
	}
	var maxStartedAt time.Time
	for _, info := range latestByName {
		if info.StartedAt.After(maxStartedAt) {
			maxStartedAt = info.StartedAt
		}
	}
	latestStatus := make(map[string]string)
	for name, info := range latestByName {
		if info.Status == "running" && info.StartedAt.Before(maxStartedAt) {
			latestStatus[name] = "completed"
		} else {
			latestStatus[name] = info.Status
		}
	}
	allCompleted := true
	anyFailed := false
	anyRunning := false
	for _, status := range latestStatus {
		if status == "failed" {
			anyFailed = true
		}
		if status != "completed" {
			allCompleted = false
		}
		if status == "running" {
			anyRunning = true
		}
	}

	resp := &PipelineRunDetailResponse{
		ExecutionID: executionID,
		StartedAt:   firstStartedAt.Format(time.RFC3339),
		Steps:       steps,
	}
	if anyFailed {
		resp.Status = "failed"
	} else if anyRunning {
		resp.Status = "running"
	} else if allCompleted {
		resp.Status = "completed"
	} else {
		resp.Status = "running"
	}
	if lastCompletedAt != nil {
		s := lastCompletedAt.Format(time.RFC3339)
		resp.CompletedAt = &s
		dur := int(lastCompletedAt.Sub(firstStartedAt).Milliseconds())
		resp.DurationMs = &dur
	}

	// Q2: table counts via ingestion_runs linked by execution_id (= pipeline_run_id)
	var tc TableCounts
	if err := r.pool.QueryRow(ctx, `
		WITH run_ids AS (
			SELECT run_id FROM pipeline.ingestion_runs WHERE pipeline_run_id = $1
		)
		SELECT
			(SELECT COUNT(*) FROM pipeline.snap_csv WHERE run_id IN (SELECT run_id FROM run_ids)),
			(SELECT COUNT(*) FROM pipeline.snap_api WHERE run_id IN (SELECT run_id FROM run_ids)),
			(SELECT COUNT(*) FROM pipeline.snap_data_quality WHERE run_id IN (SELECT run_id FROM run_ids)),
			(SELECT COUNT(*) FROM pipeline.snap_disappearances WHERE run_id IN (SELECT run_id FROM run_ids)),
			(SELECT COUNT(*) FROM pipeline.snap_reconcile_dq WHERE csv_run_id IN (SELECT run_id FROM run_ids) OR api_run_id IN (SELECT run_id FROM run_ids))
	`, executionID).Scan(
		&tc.SnapCSV, &tc.SnapAPI, &tc.SnapDataQuality, &tc.Disappearances, &tc.ReconcileDQ,
	); err != nil {
		return nil, err
	}
	resp.TableCounts = tc

	// Q3: opportunity stats
	var oppStats OpportunityStats
	if err := r.pool.QueryRow(ctx, `
		WITH run_ids AS (
			SELECT run_id FROM pipeline.ingestion_runs WHERE pipeline_run_id = $1
		)
		SELECT
			COUNT(*),
			COUNT(*) FILTER (WHERE created_at >= $2),
			COUNT(*) FILTER (WHERE created_at < $2)
		FROM opportunities
		WHERE last_csv_run_id IN (SELECT run_id FROM run_ids)
	`, executionID, firstStartedAt).Scan(
		&oppStats.TotalAffected, &oppStats.Inserted, &oppStats.Updated,
	); err != nil {
		return nil, err
	}

	if err := r.pool.QueryRow(ctx, `
		WITH run_ids AS (
			SELECT run_id FROM pipeline.ingestion_runs WHERE pipeline_run_id = $1
		)
		SELECT COUNT(DISTINCT sa.notice_id)
		FROM pipeline.snap_api sa
		WHERE sa.run_id IN (SELECT run_id FROM run_ids)
		  AND sa.notice_id IN (SELECT notice_id FROM pipeline.snap_csv WHERE run_id IN (SELECT run_id FROM run_ids))
	`, executionID).Scan(&oppStats.FromBoth); err != nil {
		return nil, err
	}
	oppStats.FromAPI = tc.SnapAPI
	oppStats.FromCSVOnly = oppStats.TotalAffected - oppStats.FromBoth
	resp.OpportunityStats = oppStats

	return resp, nil
}

// --- Snap Record Detail ---

type SnapCSVRecord struct {
	ID                 int64            `json:"id"`
	NoticeID           string           `json:"notice_id"`
	SolicitationNumber *string          `json:"solicitation_number"`
	Title              *string          `json:"title"`
	Type               *string          `json:"type"`
	BaseType           *string          `json:"base_type"`
	PostedDate         *string          `json:"posted_date"`
	ResponseDeadline   *string          `json:"response_deadline"`
	ArchiveDate        *string          `json:"archive_date"`
	ArchiveType        *string          `json:"archive_type"`
	SetAsideCode       *string          `json:"set_aside_code"`
	NaicsCode          *string          `json:"naics_code"`
	ClassificationCode *string          `json:"classification_code"`
	Active             *bool            `json:"active"`
	Department         *string          `json:"department"`
	SubTier            *string          `json:"sub_tier"`
	Office             *string          `json:"office"`
	CGAC               *string          `json:"cgac"`
	FPDSCode           *string          `json:"fpds_code"`
	AACCode            *string          `json:"aac_code"`
	AwardNumber        *string          `json:"award_number"`
	AwardDate          *string          `json:"award_date"`
	AwardAmount        *float64         `json:"award_amount"`
	RawData            json.RawMessage  `json:"raw_data"`
	ContentHash        string           `json:"content_hash"`
	RunID              string           `json:"run_id"`
	SnapshotDate       string           `json:"snapshot_date"`
	DownloadID         *int64           `json:"download_id"`
	CreatedAt          string           `json:"created_at"`
}

type SnapAPIRecord struct {
	ID           int64           `json:"id"`
	RunID        string          `json:"run_id"`
	NoticeID     string          `json:"notice_id"`
	RawData      json.RawMessage `json:"raw_data"`
	ContentHash  string          `json:"content_hash"`
	SnapshotDate string          `json:"snapshot_date"`
	CreatedAt    string          `json:"created_at"`
}

func scanSnapCSVRecord(row pgx.Row) (*SnapCSVRecord, error) {
	var r SnapCSVRecord
	var postedDate, responseDeadline, snapshotDate *time.Time
	var createdAt time.Time
	err := row.Scan(
		&r.ID, &r.NoticeID, &r.SolicitationNumber, &r.Title, &r.Type, &r.BaseType,
		&postedDate, &responseDeadline, &r.ArchiveDate, &r.ArchiveType,
		&r.SetAsideCode, &r.NaicsCode, &r.ClassificationCode, &r.Active,
		&r.Department, &r.SubTier, &r.Office, &r.CGAC, &r.FPDSCode, &r.AACCode,
		&r.AwardNumber, &r.AwardDate, &r.AwardAmount,
		&r.RawData, &r.ContentHash, &r.RunID, &snapshotDate, &r.DownloadID, &createdAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	if postedDate != nil {
		s := postedDate.Format(time.RFC3339)
		r.PostedDate = &s
	}
	if responseDeadline != nil {
		s := responseDeadline.Format(time.RFC3339)
		r.ResponseDeadline = &s
	}
	if snapshotDate != nil {
		r.SnapshotDate = snapshotDate.Format(time.RFC3339)
	}
	r.CreatedAt = createdAt.Format(time.RFC3339)
	return &r, nil
}

const snapCSVQuery = `
	SELECT id, notice_id, solicitation_number, title, type, base_type,
	       posted_date, response_deadline, archive_date, archive_type,
	       set_aside_code, naics_code, classification_code, active,
	       department, sub_tier, office, cgac, fpds_code, aac_code,
	       award_number, award_date, award_amount::float8,
	       raw_data, content_hash, run_id, snapshot_date, download_id, created_at
`

func (r *PipelineRepository) GetSnapCSVRecord(ctx context.Context, id int64) (*SnapCSVRecord, error) {
	row := r.pool.QueryRow(ctx, snapCSVQuery+` FROM pipeline.snap_csv WHERE id = $1`, id)
	return scanSnapCSVRecord(row)
}

func (r *PipelineRepository) GetSnapArchivedCSVRecord(ctx context.Context, id int64) (*SnapCSVRecord, error) {
	row := r.pool.QueryRow(ctx, snapCSVQuery+` FROM pipeline.snap_archived_csv WHERE id = $1`, id)
	return scanSnapCSVRecord(row)
}

func (r *PipelineRepository) GetSnapAPIRecord(ctx context.Context, id int64) (*SnapAPIRecord, error) {
	var rec SnapAPIRecord
	var snapshotDate, createdAt time.Time
	err := r.pool.QueryRow(ctx, `
		SELECT id, run_id, notice_id, raw_data, content_hash, snapshot_date, created_at
		FROM pipeline.snap_api WHERE id = $1
	`, id).Scan(&rec.ID, &rec.RunID, &rec.NoticeID, &rec.RawData, &rec.ContentHash, &snapshotDate, &createdAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	rec.SnapshotDate = snapshotDate.Format(time.RFC3339)
	rec.CreatedAt = createdAt.Format(time.RFC3339)
	return &rec, nil
}
