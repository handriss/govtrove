package repository

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/handriss/govtrove/api/internal/models"
)

type SavedSearchRepository struct {
	pool *pgxpool.Pool
}

func NewSavedSearchRepository(pool *pgxpool.Pool) *SavedSearchRepository {
	return &SavedSearchRepository{pool: pool}
}

func (r *SavedSearchRepository) List(ctx context.Context, userID int) ([]models.SavedSearch, error) {
	query := `SELECT id, user_id, name, filters, alert_enabled, last_checked_at, last_match_count, total_result_count, created_at, updated_at
		FROM saved_searches WHERE user_id = $1 ORDER BY updated_at DESC`

	rows, err := r.pool.Query(ctx, query, userID)
	if err != nil {
		return nil, fmt.Errorf("listing saved searches: %w", err)
	}
	defer rows.Close()

	var searches []models.SavedSearch
	for rows.Next() {
		var s models.SavedSearch
		if err := rows.Scan(&s.ID, &s.UserID, &s.Name, &s.Filters,
			&s.AlertEnabled, &s.LastCheckedAt, &s.LastMatchCount, &s.TotalResultCount,
			&s.CreatedAt, &s.UpdatedAt); err != nil {
			return nil, fmt.Errorf("scanning saved search: %w", err)
		}
		searches = append(searches, s)
	}
	return searches, rows.Err()
}

func (r *SavedSearchRepository) Create(ctx context.Context, userID int, input *models.CreateSavedSearchInput) (*models.SavedSearch, error) {
	alertEnabled := true
	if input.AlertEnabled != nil {
		alertEnabled = *input.AlertEnabled
	}

	query := `
		INSERT INTO saved_searches (user_id, name, filters, alert_enabled)
		VALUES ($1, $2, $3, $4)
		RETURNING id, user_id, name, filters, alert_enabled, last_checked_at, last_match_count, total_result_count, created_at, updated_at
	`

	var s models.SavedSearch
	err := r.pool.QueryRow(ctx, query, userID, input.Name, input.Filters, alertEnabled).Scan(
		&s.ID, &s.UserID, &s.Name, &s.Filters,
		&s.AlertEnabled, &s.LastCheckedAt, &s.LastMatchCount, &s.TotalResultCount,
		&s.CreatedAt, &s.UpdatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("creating saved search: %w", err)
	}
	return &s, nil
}

func (r *SavedSearchRepository) Update(ctx context.Context, id, userID int, input *models.UpdateSavedSearchInput) (*models.SavedSearch, error) {
	query := `
		UPDATE saved_searches
		SET name = COALESCE($3, name),
		    filters = COALESCE($4, filters),
		    alert_enabled = COALESCE($5, alert_enabled),
		    updated_at = NOW()
		WHERE id = $1 AND user_id = $2
		RETURNING id, user_id, name, filters, alert_enabled, last_checked_at, last_match_count, total_result_count, created_at, updated_at
	`

	var s models.SavedSearch
	err := r.pool.QueryRow(ctx, query, id, userID, input.Name, input.Filters, input.AlertEnabled).Scan(
		&s.ID, &s.UserID, &s.Name, &s.Filters,
		&s.AlertEnabled, &s.LastCheckedAt, &s.LastMatchCount, &s.TotalResultCount,
		&s.CreatedAt, &s.UpdatedAt,
	)
	if err == pgx.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("updating saved search: %w", err)
	}
	return &s, nil
}

func (r *SavedSearchRepository) Delete(ctx context.Context, id, userID int) error {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback(ctx)

	_, err = tx.Exec(ctx, `DELETE FROM notifications WHERE source_id = $1 AND update_type = 'search_matches'`, id)
	if err != nil {
		return fmt.Errorf("deleting related updates: %w", err)
	}

	_, err = tx.Exec(ctx, `DELETE FROM saved_searches WHERE id = $1 AND user_id = $2`, id, userID)
	if err != nil {
		return fmt.Errorf("deleting saved search: %w", err)
	}

	return tx.Commit(ctx)
}

func (r *SavedSearchRepository) GetByIDAndUser(ctx context.Context, id, userID int) (*models.SavedSearch, error) {
	var s models.SavedSearch
	err := r.pool.QueryRow(ctx,
		`SELECT id, user_id, name, filters, alert_enabled, last_checked_at, last_match_count, total_result_count, created_at, updated_at
		 FROM saved_searches WHERE id = $1 AND user_id = $2`, id, userID,
	).Scan(&s.ID, &s.UserID, &s.Name, &s.Filters,
		&s.AlertEnabled, &s.LastCheckedAt, &s.LastMatchCount, &s.TotalResultCount,
		&s.CreatedAt, &s.UpdatedAt)
	if err == pgx.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("fetching saved search: %w", err)
	}
	return &s, nil
}

func (r *SavedSearchRepository) HistoryTimeline(ctx context.Context, id, userID, days int) (*models.HistoryTimelineResponse, error) {
	search, err := r.GetByIDAndUser(ctx, id, userID)
	if err != nil {
		return nil, err
	}
	if search == nil {
		return nil, nil
	}

	params, err := ParseSavedFilters(search.Filters)
	if err != nil {
		return nil, fmt.Errorf("parsing filters: %w", err)
	}

	conditions, args, argNum, _ := buildFilterConditions(params, "", 1)

	now := time.Now().UTC()
	endDate := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.UTC).AddDate(0, 0, 1)
	startDate := endDate.AddDate(0, 0, -days)

	conditions = append(conditions, fmt.Sprintf("created_at >= $%d", argNum))
	args = append(args, startDate)
	argNum++
	conditions = append(conditions, fmt.Sprintf("created_at < $%d", argNum))
	args = append(args, endDate)

	where := strings.Join(conditions, " AND ")
	query := fmt.Sprintf(`
		SELECT DATE(created_at) AS day, COUNT(*)
		FROM opportunities
		WHERE %s
		GROUP BY DATE(created_at)
		ORDER BY day DESC
	`, where)

	rows, err := r.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("querying history timeline: %w", err)
	}
	defer rows.Close()

	countMap := make(map[string]int)
	for rows.Next() {
		var day time.Time
		var count int
		if err := rows.Scan(&day, &count); err != nil {
			return nil, fmt.Errorf("scanning timeline row: %w", err)
		}
		countMap[day.Format("2006-01-02")] = count
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterating timeline rows: %w", err)
	}

	result := &models.HistoryTimelineResponse{Search: *search}
	for i := 0; i < days; i++ {
		date := endDate.AddDate(0, 0, -1-i).Format("2006-01-02")
		count := countMap[date]
		result.Days = append(result.Days, models.DayCount{Date: date, Count: count})
		result.TotalNew += count
		if count > 0 {
			result.DaysWithMatches++
		}
	}

	return result, nil
}

func (r *SavedSearchRepository) HistoryDay(ctx context.Context, id, userID int, date string, page, limit int) (*models.SavedSearch, *models.SearchResult, error) {
	search, err := r.GetByIDAndUser(ctx, id, userID)
	if err != nil {
		return nil, nil, err
	}
	if search == nil {
		return nil, nil, nil
	}

	params, err := ParseSavedFilters(search.Filters)
	if err != nil {
		return nil, nil, fmt.Errorf("parsing filters: %w", err)
	}

	conditions, args, argNum, _ := buildFilterConditions(params, "", 1)

	dayStart, err := time.Parse("2006-01-02", date)
	if err != nil {
		return nil, nil, fmt.Errorf("invalid date: %w", err)
	}
	dayEnd := dayStart.AddDate(0, 0, 1)

	conditions = append(conditions, fmt.Sprintf("created_at >= $%d", argNum))
	args = append(args, dayStart)
	argNum++
	conditions = append(conditions, fmt.Sprintf("created_at < $%d", argNum))
	args = append(args, dayEnd)
	argNum++

	where := strings.Join(conditions, " AND ")

	var total int
	countQuery := fmt.Sprintf("SELECT COUNT(*) FROM opportunities WHERE %s", where)
	if err := r.pool.QueryRow(ctx, countQuery, args...).Scan(&total); err != nil {
		return nil, nil, fmt.Errorf("counting history day: %w", err)
	}

	offset := (page - 1) * limit
	dataArgs := append([]any{}, args...)
	dataArgs = append(dataArgs, limit, offset)
	dataQuery := fmt.Sprintf(`
		SELECT id, notice_id, title, description, solicitation_number, type,
			department, posted_date, response_deadline,
			set_aside_code, set_aside_description, naics_code,
			pop_state, active
		FROM opportunities
		WHERE %s
		ORDER BY created_at DESC
		LIMIT $%d OFFSET $%d
	`, where, argNum, argNum+1)

	rows, err := r.pool.Query(ctx, dataQuery, dataArgs...)
	if err != nil {
		return nil, nil, fmt.Errorf("querying history day: %w", err)
	}
	defer rows.Close()

	var opps []models.OpportunityListItem
	for rows.Next() {
		var opp models.OpportunityListItem
		if err := rows.Scan(
			&opp.ID, &opp.NoticeID, &opp.Title, &opp.Description,
			&opp.SolicitationNumber, &opp.Type, &opp.Department,
			&opp.PostedDate, &opp.ResponseDeadline,
			&opp.SetAsideCode, &opp.SetAsideDesc, &opp.NAICSCode,
			&opp.PopState, &opp.Active,
		); err != nil {
			return nil, nil, fmt.Errorf("scanning history day row: %w", err)
		}
		opps = append(opps, opp)
	}

	totalPages := (total + limit - 1) / limit
	if totalPages == 0 {
		totalPages = 1
	}

	if opps == nil {
		opps = []models.OpportunityListItem{}
	}

	return search, &models.SearchResult{
		Opportunities: opps,
		Total:         total,
		Page:          page,
		Limit:         limit,
		TotalPages:    totalPages,
	}, nil
}

func (r *SavedSearchRepository) Run(ctx context.Context, id, userID int) (*models.SavedSearch, *models.SearchResult, error) {
	var s models.SavedSearch
	err := r.pool.QueryRow(ctx,
		`SELECT id, user_id, name, filters, alert_enabled, last_checked_at, last_match_count, total_result_count, created_at, updated_at
		 FROM saved_searches WHERE id = $1 AND user_id = $2`, id, userID,
	).Scan(&s.ID, &s.UserID, &s.Name, &s.Filters,
		&s.AlertEnabled, &s.LastCheckedAt, &s.LastMatchCount, &s.TotalResultCount,
		&s.CreatedAt, &s.UpdatedAt)
	if err == pgx.ErrNoRows {
		return nil, nil, nil
	}
	if err != nil {
		return nil, nil, fmt.Errorf("fetching saved search: %w", err)
	}

	params, err := ParseSavedFilters(s.Filters)
	if err != nil {
		return nil, nil, fmt.Errorf("parsing filters: %w", err)
	}
	params.Page = 1
	params.Limit = 25

	oppRepo := NewOpportunityRepository(r.pool)
	result, err := oppRepo.Search(ctx, params)
	if err != nil {
		return nil, nil, fmt.Errorf("running search: %w", err)
	}

	totalCount := result.Total
	_, err = r.pool.Exec(ctx,
		`UPDATE saved_searches SET total_result_count = $1, updated_at = NOW() WHERE id = $2`,
		totalCount, s.ID)
	if err != nil {
		return nil, nil, fmt.Errorf("updating total_result_count: %w", err)
	}
	s.TotalResultCount = &totalCount

	return &s, result, nil
}
