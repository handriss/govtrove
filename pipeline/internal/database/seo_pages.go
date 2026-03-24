package database

import (
	"context"
	"fmt"
	"time"
)

type SEOPageCount struct {
	FilterType  string
	FilterValue string
	TotalCount  int
	RecentCount int
}

type SEOOpportunity struct {
	NoticeID           string
	Title              string
	Department         string
	NAICSCode          *string
	SetAsideCode       *string
	PostedDate         *time.Time
	ResponseDeadline   *time.Time
	PopState           *string
	SolicitationNumber *string
}

type AgencyInfo struct {
	Department string
	TotalCount int
}

func (db *DB) GetSEOPageCounts(ctx context.Context, recentSince time.Time) ([]SEOPageCount, error) {
	rows, err := db.pool.Query(ctx, `
		SELECT 'naics' AS filter_type, naics_code AS filter_value,
		       COUNT(*) AS total_count,
		       COUNT(*) FILTER (WHERE posted_date >= $1) AS recent_count
		FROM opportunities WHERE active = true AND is_latest = true
		  AND naics_code IS NOT NULL AND naics_code != ''
		GROUP BY naics_code
		UNION ALL
		SELECT 'set_aside', set_aside_code, COUNT(*),
		       COUNT(*) FILTER (WHERE posted_date >= $1)
		FROM opportunities WHERE active = true AND is_latest = true
		  AND set_aside_code IS NOT NULL AND set_aside_code != ''
		GROUP BY set_aside_code
		UNION ALL
		SELECT 'department', department, COUNT(*),
		       COUNT(*) FILTER (WHERE posted_date >= $1)
		FROM opportunities WHERE active = true AND is_latest = true
		  AND department IS NOT NULL AND department != ''
		GROUP BY department
	`, recentSince)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var results []SEOPageCount
	for rows.Next() {
		var r SEOPageCount
		if err := rows.Scan(&r.FilterType, &r.FilterValue, &r.TotalCount, &r.RecentCount); err != nil {
			return nil, err
		}
		results = append(results, r)
	}
	return results, rows.Err()
}

func (db *DB) GetSEOPageOpportunities(ctx context.Context, filterCol, filterVal string, limit int) ([]SEOOpportunity, error) {
	allowed := map[string]bool{"naics_code": true, "set_aside_code": true, "department": true}
	if !allowed[filterCol] {
		return nil, fmt.Errorf("invalid filter column: %s", filterCol)
	}

	query := fmt.Sprintf(`
		SELECT notice_id, title, department, naics_code, set_aside_code,
		       posted_date, response_deadline, pop_state, solicitation_number
		FROM opportunities
		WHERE %s = $1 AND active = true AND is_latest = true
		ORDER BY posted_date DESC LIMIT $2
	`, filterCol)

	rows, err := db.pool.Query(ctx, query, filterVal, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var results []SEOOpportunity
	for rows.Next() {
		var r SEOOpportunity
		if err := rows.Scan(&r.NoticeID, &r.Title, &r.Department, &r.NAICSCode, &r.SetAsideCode,
			&r.PostedDate, &r.ResponseDeadline, &r.PopState, &r.SolicitationNumber); err != nil {
			return nil, err
		}
		results = append(results, r)
	}
	return results, rows.Err()
}

func (db *DB) GetActiveAgencies(ctx context.Context) ([]AgencyInfo, error) {
	rows, err := db.pool.Query(ctx, `
		SELECT department, COUNT(*) as cnt FROM opportunities
		WHERE active = true AND is_latest = true
		  AND department IS NOT NULL AND department != ''
		GROUP BY department ORDER BY cnt DESC
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var results []AgencyInfo
	for rows.Next() {
		var r AgencyInfo
		if err := rows.Scan(&r.Department, &r.TotalCount); err != nil {
			return nil, err
		}
		results = append(results, r)
	}
	return results, rows.Err()
}
