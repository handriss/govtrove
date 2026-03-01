package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"strings"
	"time"

	awsconfig "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/secretsmanager"
	"github.com/aws/aws-lambda-go/lambda"
	"github.com/getsentry/sentry-go"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/handriss/govtrove/pipeline/internal/database"
)

var (
	db     *database.DB
	pool   *pgxpool.Pool
	logger *slog.Logger
)

func init() {
	ctx := context.Background()
	logger = slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo}))
	slog.SetDefault(logger)

	secretARN := os.Getenv("DATABASE_URL_SECRET_ARN")
	if secretARN == "" {
		return
	}

	if dsn := os.Getenv("SENTRY_DSN"); dsn != "" {
		sentry.Init(sentry.ClientOptions{
			Dsn:              dsn,
			Environment:      "production",
			AttachStacktrace: true,
		})
	}

	region := os.Getenv("AWS_REGION_NAME")
	if region == "" {
		region = "us-east-1"
	}

	awsCfg, err := awsconfig.LoadDefaultConfig(ctx, awsconfig.WithRegion(region))
	if err != nil {
		logger.Error("failed to load AWS config", "error", err)
		os.Exit(1)
	}

	smClient := secretsmanager.NewFromConfig(awsCfg)
	result, err := smClient.GetSecretValue(ctx, &secretsmanager.GetSecretValueInput{
		SecretId: &secretARN,
	})
	if err != nil {
		logger.Error("failed to get database URL from Secrets Manager", "error", err)
		os.Exit(1)
	}

	db, err = database.New(ctx, *result.SecretString)
	if err != nil {
		logger.Error("failed to connect to database", "error", err)
		os.Exit(1)
	}
	pool = db.Pool()

	logger.Info("cold start complete")
}

type Input struct {
	ExecutionID string `json:"execution_id"`
}

type Output struct {
	Status            string `json:"status"`
	SearchAlerts      int    `json:"search_alerts"`
	OpportunityAlerts int    `json:"opportunity_alerts"`
}

type Handler struct {
	Pool   *pgxpool.Pool
	Store  database.Store
	Logger *slog.Logger
}

func (h *Handler) Handle(ctx context.Context, event json.RawMessage) (_ *Output, retErr error) {
	defer func() {
		if retErr != nil {
			sentry.CaptureException(retErr)
		}
		sentry.Flush(2 * time.Second)
	}()

	start := time.Now()

	// Parse execution ID from input
	var input Input
	if err := json.Unmarshal(event, &input); err != nil {
		h.Logger.Warn("failed to parse input", "error", err)
	}

	var stepID uuid.UUID
	if input.ExecutionID != "" {
		execID, err := uuid.Parse(input.ExecutionID)
		if err == nil && h.Store != nil {
			sid, err := h.Store.CreatePipelineStep(ctx, execID, "generate-alerts")
			if err != nil {
				h.Logger.Warn("failed to create pipeline step", "error", err)
			} else {
				stepID = sid
			}
		}
	}

	searchAlerts, err := h.processSearchAlerts(ctx)
	if err != nil {
		h.Logger.Error("search alerts failed", "error", err)
		if stepID != uuid.Nil {
			_ = h.Store.FailPipelineStep(ctx, stepID, err.Error(), int(time.Since(start).Milliseconds()))
		}
		return nil, fmt.Errorf("search alerts: %w", err)
	}

	oppAlerts, err := h.processOpportunityAlerts(ctx)
	if err != nil {
		h.Logger.Error("opportunity alerts failed", "error", err)
		if stepID != uuid.Nil {
			_ = h.Store.FailPipelineStep(ctx, stepID, err.Error(), int(time.Since(start).Milliseconds()))
		}
		return nil, fmt.Errorf("opportunity alerts: %w", err)
	}

	durationMs := int(time.Since(start).Milliseconds())

	if stepID != uuid.Nil {
		stepStats := map[string]any{
			"search_alerts":      searchAlerts,
			"opportunity_alerts": oppAlerts,
		}
		if err := h.Store.CompletePipelineStep(ctx, stepID, stepStats, durationMs); err != nil {
			h.Logger.Warn("failed to complete pipeline step", "error", err)
		}
	}

	h.Logger.Info("alerts complete",
		"search_alerts", searchAlerts,
		"opportunity_alerts", oppAlerts,
		"duration_ms", durationMs,
	)

	return &Output{
		Status:            "ok",
		SearchAlerts:      searchAlerts,
		OpportunityAlerts: oppAlerts,
	}, nil
}

// --- Saved Search Alerts ---

type savedSearchRow struct {
	ID            int
	UserID        int
	Filters       json.RawMessage
	LastCheckedAt *time.Time
}

func (h *Handler) processSearchAlerts(ctx context.Context) (int, error) {
	if !h.acquireLock(ctx, "saved_search_alerts") {
		h.Logger.Info("saved_search_alerts lock held, skipping")
		return 0, nil
	}
	defer h.releaseLock(ctx, "saved_search_alerts")

	totalAlerts := 0
	offset := 0
	batchSize := 100

	for {
		rows, err := h.Pool.Query(ctx, `
			SELECT id, user_id, filters, last_checked_at
			FROM saved_searches
			WHERE alert_enabled = true
			ORDER BY id
			LIMIT $1 OFFSET $2
		`, batchSize, offset)
		if err != nil {
			return totalAlerts, fmt.Errorf("query saved searches: %w", err)
		}

		var searches []savedSearchRow
		for rows.Next() {
			var s savedSearchRow
			if err := rows.Scan(&s.ID, &s.UserID, &s.Filters, &s.LastCheckedAt); err != nil {
				rows.Close()
				return totalAlerts, fmt.Errorf("scan saved search: %w", err)
			}
			searches = append(searches, s)
		}
		rows.Close()

		if len(searches) == 0 {
			break
		}

		for _, s := range searches {
			created, err := h.checkSearchForNewMatches(ctx, s)
			if err != nil {
				h.Logger.Error("check search failed", "search_id", s.ID, "error", err)
				continue
			}
			if created {
				totalAlerts++
			}
		}

		offset += batchSize
	}

	return totalAlerts, nil
}

type savedFilters struct {
	Keyword        string   `json:"keyword"`
	ExactMatch     bool     `json:"exactMatch"`
	NAICS          []string `json:"naics"`
	PSC            []string `json:"psc"`
	SetAside       []string `json:"setAside"`
	Department     string   `json:"department"`
	State          string   `json:"state"`
	NoticeType     []string `json:"noticeType"`
	DeadlinePreset string   `json:"deadlinePreset"`
	PostedFrom     string   `json:"postedFrom"`
	PostedTo       string   `json:"postedTo"`
	DeadlineFrom   string   `json:"deadlineFrom"`
	DeadlineTo     string   `json:"deadlineTo"`
}

func (h *Handler) checkSearchForNewMatches(ctx context.Context, s savedSearchRow) (bool, error) {
	var f savedFilters
	if err := json.Unmarshal(s.Filters, &f); err != nil {
		return false, fmt.Errorf("unmarshal filters: %w", err)
	}

	conditions := []string{"active = true", "is_latest = true"}
	args := []any{}
	argNum := 1

	// Only match opportunities created since last check
	if s.LastCheckedAt != nil {
		conditions = append(conditions, fmt.Sprintf("created_at > $%d", argNum))
		args = append(args, *s.LastCheckedAt)
		argNum++
	}

	conditions, args, argNum = appendFilterConditions(conditions, args, argNum, f)

	// Count new matches
	query := fmt.Sprintf("SELECT COUNT(*) FROM opportunities WHERE %s", strings.Join(conditions, " AND "))
	var matchCount int
	if err := h.Pool.QueryRow(ctx, query, args...).Scan(&matchCount); err != nil {
		return false, fmt.Errorf("count matches: %w", err)
	}

	// Also get total result count (ignoring the created_at constraint)
	totalConditions := []string{"active = true", "is_latest = true"}
	totalArgs := []any{}
	totalArgNum := 1
	totalConditions, totalArgs, _ = appendFilterConditions(totalConditions, totalArgs, totalArgNum, f)
	totalQuery := fmt.Sprintf("SELECT COUNT(*) FROM opportunities WHERE %s", strings.Join(totalConditions, " AND "))
	var totalCount int
	if err := h.Pool.QueryRow(ctx, totalQuery, totalArgs...).Scan(&totalCount); err != nil {
		return false, fmt.Errorf("count total: %w", err)
	}

	now := time.Now()
	_, err := h.Pool.Exec(ctx, `
		UPDATE saved_searches
		SET last_checked_at = $2, last_match_count = $3, total_result_count = $4
		WHERE id = $1
	`, s.ID, now, matchCount, totalCount)
	if err != nil {
		return false, fmt.Errorf("update search: %w", err)
	}

	if matchCount == 0 {
		return false, nil
	}

	// Grab IDs for the new matches (cap at 50)
	idsQuery := fmt.Sprintf("SELECT id FROM opportunities WHERE %s ORDER BY posted_date DESC LIMIT 50", strings.Join(conditions, " AND "))
	idsRows, err := h.Pool.Query(ctx, idsQuery, args...)
	if err != nil {
		return false, fmt.Errorf("query match ids: %w", err)
	}
	defer idsRows.Close()

	var oppIDs []int
	for idsRows.Next() {
		var id int
		if err := idsRows.Scan(&id); err != nil {
			return false, fmt.Errorf("scan match id: %w", err)
		}
		oppIDs = append(oppIDs, id)
	}

	summary := fmt.Sprintf("%d new opportunities match your saved search", matchCount)
	if matchCount == 1 {
		summary = "1 new opportunity matches your saved search"
	}

	_, err = h.Pool.Exec(ctx, `
		INSERT INTO user_updates (id, user_id, update_type, source_id, opportunity_ids, summary, details)
		VALUES ($1, $2, 'saved_search_matches', $3, $4, $5, $6)
	`, uuid.New(), s.UserID, s.ID, oppIDs, summary, nil)
	if err != nil {
		return false, fmt.Errorf("insert update: %w", err)
	}

	return true, nil
}

func appendFilterConditions(conditions []string, args []any, argNum int, f savedFilters) ([]string, []any, int) {
	if f.Keyword != "" {
		if f.ExactMatch {
			conditions = append(conditions, fmt.Sprintf("(title ILIKE $%d OR description ILIKE $%d OR solicitation_number ILIKE $%d)", argNum, argNum, argNum))
			args = append(args, "%"+f.Keyword+"%")
			argNum++
		} else {
			conditions = append(conditions, fmt.Sprintf("search_vector @@ websearch_to_tsquery('english', $%d)", argNum))
			args = append(args, f.Keyword)
			argNum++
		}
	}

	if len(f.NoticeType) > 0 {
		conditions = append(conditions, fmt.Sprintf("type = ANY($%d)", argNum))
		args = append(args, f.NoticeType)
		argNum++
	}

	if len(f.SetAside) > 0 {
		conditions = append(conditions, fmt.Sprintf("set_aside_code = ANY($%d)", argNum))
		args = append(args, f.SetAside)
		argNum++
	}

	if len(f.NAICS) > 0 {
		conditions = append(conditions, fmt.Sprintf("naics_code = ANY($%d)", argNum))
		args = append(args, f.NAICS)
		argNum++
	}

	if len(f.PSC) > 0 {
		conditions = append(conditions, fmt.Sprintf("classification_code = ANY($%d)", argNum))
		args = append(args, f.PSC)
		argNum++
	}

	if f.Department != "" {
		conditions = append(conditions, fmt.Sprintf("department ILIKE $%d", argNum))
		args = append(args, "%"+f.Department+"%")
		argNum++
	}

	if f.State != "" {
		conditions = append(conditions, fmt.Sprintf("pop_state = $%d", argNum))
		args = append(args, f.State)
		argNum++
	}

	now := time.Now()

	if f.PostedFrom != "" {
		if t, err := time.Parse("2006-01-02", f.PostedFrom); err == nil {
			conditions = append(conditions, fmt.Sprintf("posted_date >= $%d", argNum))
			args = append(args, t)
			argNum++
		}
	}
	if f.PostedTo != "" {
		if t, err := time.Parse("2006-01-02", f.PostedTo); err == nil {
			conditions = append(conditions, fmt.Sprintf("posted_date <= $%d", argNum))
			args = append(args, t)
			argNum++
		}
	}

	if f.DeadlinePreset != "" {
		today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.UTC)
		conditions = append(conditions, fmt.Sprintf("response_deadline >= $%d", argNum))
		args = append(args, today)
		argNum++
		if to := deadlinePresetToDate(f.DeadlinePreset, now); to != nil {
			conditions = append(conditions, fmt.Sprintf("response_deadline <= $%d", argNum))
			args = append(args, *to)
			argNum++
		}
	} else {
		if f.DeadlineFrom != "" {
			if t, err := time.Parse("2006-01-02", f.DeadlineFrom); err == nil {
				conditions = append(conditions, fmt.Sprintf("response_deadline >= $%d", argNum))
				args = append(args, t)
				argNum++
			}
		}
		if f.DeadlineTo != "" {
			if t, err := time.Parse("2006-01-02", f.DeadlineTo); err == nil {
				conditions = append(conditions, fmt.Sprintf("response_deadline <= $%d", argNum))
				args = append(args, t)
				argNum++
			}
		}
	}

	return conditions, args, argNum
}

func deadlinePresetToDate(preset string, now time.Time) *time.Time {
	switch preset {
	case "7":
		t := now.AddDate(0, 0, 7)
		return &t
	case "14":
		t := now.AddDate(0, 0, 14)
		return &t
	case "30":
		t := now.AddDate(0, 0, 30)
		return &t
	case "60":
		t := now.AddDate(0, 0, 60)
		return &t
	case "90":
		t := now.AddDate(0, 0, 90)
		return &t
	case "quarter":
		endMonth := ((int(now.Month())-1)/3+1)*3 + 1
		year := now.Year()
		if endMonth > 12 {
			endMonth -= 12
			year++
		}
		t := time.Date(year, time.Month(endMonth), 0, 0, 0, 0, 0, time.UTC)
		return &t
	}
	return nil
}

// --- Opportunity Change Alerts ---

func (h *Handler) processOpportunityAlerts(ctx context.Context) (int, error) {
	if !h.acquireLock(ctx, "opportunity_change_alerts") {
		h.Logger.Info("opportunity_change_alerts lock held, skipping")
		return 0, nil
	}
	defer h.releaseLock(ctx, "opportunity_change_alerts")

	totalAlerts := 0

	// Amendments — new notice under same solicitation
	amendments, err := h.detectAmendments(ctx)
	if err != nil {
		return 0, fmt.Errorf("detect amendments: %w", err)
	}
	totalAlerts += amendments

	// In-place changes detected via versioned rows
	changes, err := h.detectInPlaceChanges(ctx)
	if err != nil {
		return totalAlerts, fmt.Errorf("detect changes: %w", err)
	}
	totalAlerts += changes

	return totalAlerts, nil
}

type amendmentRow struct {
	SavedOppID int
	UserID     int
	SolNumber  string
	NewOppID   int
	NewNoticeID string
	Title      string
}

func (h *Handler) detectAmendments(ctx context.Context) (int, error) {
	rows, err := h.Pool.Query(ctx, `
		SELECT so.id, so.user_id, so.solicitation_number, o.id, o.notice_id, o.title
		FROM saved_opportunities so
		JOIN opportunities o ON o.solicitation_number = so.solicitation_number
		WHERE o.notice_id != so.notice_id
		  AND o.created_at > so.last_notified_at
		  AND so.solicitation_number IS NOT NULL
		  AND o.is_latest = true
	`)
	if err != nil {
		return 0, fmt.Errorf("query amendments: %w", err)
	}
	defer rows.Close()

	var amendments []amendmentRow
	for rows.Next() {
		var a amendmentRow
		if err := rows.Scan(&a.SavedOppID, &a.UserID, &a.SolNumber, &a.NewOppID, &a.NewNoticeID, &a.Title); err != nil {
			return 0, fmt.Errorf("scan amendment: %w", err)
		}
		amendments = append(amendments, a)
	}

	count := 0
	for _, a := range amendments {
		summary := fmt.Sprintf("New amendment posted for %s: %s", a.SolNumber, a.Title)
		details, _ := json.Marshal(map[string]any{
			"type":              "amendment",
			"solicitation_number": a.SolNumber,
			"new_notice_id":     a.NewNoticeID,
			"new_opportunity_id": a.NewOppID,
		})

		_, err := h.Pool.Exec(ctx, `
			INSERT INTO user_updates (id, user_id, update_type, source_id, opportunity_ids, summary, details)
			VALUES ($1, $2, 'opportunity_amended', $3, $4, $5, $6)
		`, uuid.New(), a.UserID, a.SavedOppID, []int{a.NewOppID}, summary, string(details))
		if err != nil {
			h.Logger.Error("insert amendment update", "error", err)
			continue
		}

		h.Pool.Exec(ctx, `UPDATE saved_opportunities SET last_notified_at = NOW() WHERE id = $1`, a.SavedOppID)
		count++
	}

	return count, nil
}

type versionChangeRow struct {
	SavedOppID int
	UserID     int
	NoticeID   string
	// New version fields
	NewTitle             *string
	NewResponseDeadline  *time.Time
	NewArchiveDate       *time.Time
	NewDescription       *string
	NewSetAsideCode      *string
	NewAwardAmount       *float64
	NewResourceLinks     *string
	// Previous version fields
	OldTitle             *string
	OldResponseDeadline  *time.Time
	OldArchiveDate       *time.Time
	OldDescription       *string
	OldSetAsideCode      *string
	OldAwardAmount       *float64
	OldResourceLinks     *string
}

func (h *Handler) detectInPlaceChanges(ctx context.Context) (int, error) {
	// Find saved opportunities where a new version was created since last_notified_at.
	// Join the latest version (curr) against the previous version (prev) to diff fields.
	rows, err := h.Pool.Query(ctx, `
		SELECT so.id, so.user_id, so.notice_id,
			curr.title, curr.response_deadline, curr.archive_date, curr.description, curr.set_aside_code, curr.award_amount, curr.resource_links,
			prev.title, prev.response_deadline, prev.archive_date, prev.description, prev.set_aside_code, prev.award_amount, prev.resource_links
		FROM saved_opportunities so
		JOIN opportunities curr ON curr.notice_id = so.notice_id AND curr.is_latest = true AND curr.version > 1
		JOIN opportunities prev ON prev.notice_id = so.notice_id AND prev.version = curr.version - 1
		WHERE curr.created_at > so.last_notified_at
	`)
	if err != nil {
		return 0, fmt.Errorf("query version changes: %w", err)
	}
	defer rows.Close()

	count := 0
	for rows.Next() {
		var r versionChangeRow
		if err := rows.Scan(
			&r.SavedOppID, &r.UserID, &r.NoticeID,
			&r.NewTitle, &r.NewResponseDeadline, &r.NewArchiveDate, &r.NewDescription, &r.NewSetAsideCode, &r.NewAwardAmount, &r.NewResourceLinks,
			&r.OldTitle, &r.OldResponseDeadline, &r.OldArchiveDate, &r.OldDescription, &r.OldSetAsideCode, &r.OldAwardAmount, &r.OldResourceLinks,
		); err != nil {
			return count, fmt.Errorf("scan version change: %w", err)
		}

		var fieldNames []string
		var diffs []map[string]any

		if !strPtrEqual(r.OldTitle, r.NewTitle) {
			fieldNames = append(fieldNames, "Title")
			diffs = append(diffs, diffPtrs("title", r.OldTitle, r.NewTitle))
		}
		if !timePtrEqual(r.OldResponseDeadline, r.NewResponseDeadline) {
			fieldNames = append(fieldNames, friendlyFieldName("response_deadline"))
			diffs = append(diffs, diffTimePtrs("response_deadline", r.OldResponseDeadline, r.NewResponseDeadline))
		}
		if !timePtrEqual(r.OldArchiveDate, r.NewArchiveDate) {
			fieldNames = append(fieldNames, friendlyFieldName("archive_date"))
			diffs = append(diffs, diffTimePtrs("archive_date", r.OldArchiveDate, r.NewArchiveDate))
		}
		if !strPtrEqual(r.OldDescription, r.NewDescription) {
			fieldNames = append(fieldNames, friendlyFieldName("description"))
			diffs = append(diffs, map[string]any{"field": "description"})
		}
		if !strPtrEqual(r.OldSetAsideCode, r.NewSetAsideCode) {
			fieldNames = append(fieldNames, friendlyFieldName("set_aside_code"))
			diffs = append(diffs, diffPtrs("set_aside_code", r.OldSetAsideCode, r.NewSetAsideCode))
		}
		if !floatPtrEqual(r.OldAwardAmount, r.NewAwardAmount) {
			fieldNames = append(fieldNames, friendlyFieldName("award_amount"))
			diffs = append(diffs, diffFloatPtrs("award_amount", r.OldAwardAmount, r.NewAwardAmount))
		}
		if added, removed := diffResourceLinks(r.OldResourceLinks, r.NewResourceLinks); len(added) > 0 || len(removed) > 0 {
			parts := []string{}
			if len(added) > 0 {
				parts = append(parts, fmt.Sprintf("%d added", len(added)))
			}
			if len(removed) > 0 {
				parts = append(parts, fmt.Sprintf("%d removed", len(removed)))
			}
			fieldNames = append(fieldNames, fmt.Sprintf("Attachments (%s)", strings.Join(parts, ", ")))
			diffs = append(diffs, map[string]any{
				"field":   "resource_links",
				"added":   added,
				"removed": removed,
			})
		}

		if len(fieldNames) == 0 {
			continue
		}

		summary := fmt.Sprintf("Changes detected: %s", strings.Join(fieldNames, ", "))
		details, _ := json.Marshal(map[string]any{
			"type":    "field_changes",
			"changes": diffs,
		})

		_, err := h.Pool.Exec(ctx, `
			INSERT INTO user_updates (id, user_id, update_type, source_id, summary, details)
			VALUES ($1, $2, 'opportunity_changed', $3, $4, $5)
		`, uuid.New(), r.UserID, r.SavedOppID, summary, string(details))
		if err != nil {
			h.Logger.Error("insert change update", "error", err)
			continue
		}

		h.Pool.Exec(ctx, `UPDATE saved_opportunities SET last_notified_at = NOW() WHERE id = $1`, r.SavedOppID)
		count++
	}

	return count, nil
}

func strPtrEqual(a, b *string) bool {
	if a == nil && b == nil {
		return true
	}
	if a == nil || b == nil {
		return false
	}
	return *a == *b
}

func timePtrEqual(a, b *time.Time) bool {
	if a == nil && b == nil {
		return true
	}
	if a == nil || b == nil {
		return false
	}
	return a.Equal(*b)
}

func floatPtrEqual(a, b *float64) bool {
	if a == nil && b == nil {
		return true
	}
	if a == nil || b == nil {
		return false
	}
	return *a == *b
}

func diffPtrs(field string, old, new *string) map[string]any {
	d := map[string]any{"field": field}
	if old != nil {
		d["old"] = *old
	}
	if new != nil {
		d["new"] = *new
	}
	return d
}

func diffTimePtrs(field string, old, new *time.Time) map[string]any {
	d := map[string]any{"field": field}
	if old != nil {
		d["old"] = old.Format(time.RFC3339)
	}
	if new != nil {
		d["new"] = new.Format(time.RFC3339)
	}
	return d
}

func diffFloatPtrs(field string, old, new *float64) map[string]any {
	d := map[string]any{"field": field}
	if old != nil {
		d["old"] = fmt.Sprintf("%.2f", *old)
	}
	if new != nil {
		d["new"] = fmt.Sprintf("%.2f", *new)
	}
	return d
}

func diffResourceLinks(oldJSON, newJSON *string) (added, removed []string) {
	var oldLinks, newLinks []string
	if oldJSON != nil {
		json.Unmarshal([]byte(*oldJSON), &oldLinks)
	}
	if newJSON != nil {
		json.Unmarshal([]byte(*newJSON), &newLinks)
	}
	oldSet := make(map[string]bool, len(oldLinks))
	for _, l := range oldLinks {
		oldSet[l] = true
	}
	newSet := make(map[string]bool, len(newLinks))
	for _, l := range newLinks {
		newSet[l] = true
	}
	for _, l := range newLinks {
		if !oldSet[l] {
			added = append(added, l)
		}
	}
	for _, l := range oldLinks {
		if !newSet[l] {
			removed = append(removed, l)
		}
	}
	return added, removed
}

func friendlyFieldName(field string) string {
	switch field {
	case "response_deadline":
		return "Response Deadline"
	case "archive_date":
		return "Archive Date"
	case "description":
		return "Description"
	case "set_aside_code":
		return "Set-Aside"
	case "award_amount":
		return "Award Amount"
	default:
		return field
	}
}

// --- Lock mechanism ---

func (h *Handler) acquireLock(ctx context.Context, jobName string) bool {
	tag, err := h.Pool.Exec(ctx, `
		UPDATE alert_jobs
		SET locked_at = NOW(), locked_by = $2
		WHERE job_name = $1
		  AND (locked_at IS NULL OR locked_at < NOW() - interval '10 minutes')
	`, jobName, "generate-alerts-lambda")
	if err != nil {
		h.Logger.Error("acquire lock failed", "job", jobName, "error", err)
		return false
	}
	return tag.RowsAffected() > 0
}

func (h *Handler) releaseLock(ctx context.Context, jobName string) {
	_, err := h.Pool.Exec(ctx, `
		UPDATE alert_jobs
		SET locked_at = NULL, locked_by = NULL, last_run_at = NOW()
		WHERE job_name = $1
	`, jobName)
	if err != nil {
		h.Logger.Error("release lock failed", "job", jobName, "error", err)
	}
}

func main() {
	h := &Handler{Pool: pool, Store: db, Logger: logger}
	lambda.Start(h.Handle)
}
