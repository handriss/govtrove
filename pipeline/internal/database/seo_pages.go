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
		UNION ALL
		SELECT 'psc', classification_code, COUNT(*),
		       COUNT(*) FILTER (WHERE posted_date >= $1)
		FROM opportunities WHERE active = true AND is_latest = true
		  AND LENGTH(classification_code) = 4
		GROUP BY classification_code
		UNION ALL
		-- Group counts aggregate by prefix so an opportunity classified directly on the
		-- 2-character group counts alongside its 4-character children.
		SELECT 'psc_group', LEFT(classification_code, 2), COUNT(*),
		       COUNT(*) FILTER (WHERE posted_date >= $1)
		FROM opportunities WHERE active = true AND is_latest = true
		  AND LENGTH(classification_code) >= 2
		GROUP BY LEFT(classification_code, 2)
		UNION ALL
		-- Service codes hang off a single-letter category rather than a 2-character group.
		SELECT 'psc_cat', LEFT(classification_code, 1), COUNT(*),
		       COUNT(*) FILTER (WHERE posted_date >= $1)
		FROM opportunities WHERE active = true AND is_latest = true
		  AND classification_code IS NOT NULL AND classification_code != ''
		GROUP BY LEFT(classification_code, 1)
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
	allowed := map[string]bool{"naics_code": true, "set_aside_code": true, "department": true, "classification_code": true}
	if !allowed[filterCol] {
		return nil, fmt.Errorf("invalid filter column: %s", filterCol)
	}

	// COALESCE the non-nilable text columns: department (and defensively title) can be
	// NULL for some opportunities, which crashes the row scan into a *string and made
	// whole NAICS pages fail to generate.
	query := fmt.Sprintf(`
		SELECT notice_id, COALESCE(title, '') AS title, COALESCE(department, '') AS department,
		       naics_code, set_aside_code,
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

// GetSEOPageOpportunitiesByPrefix backs the PSC group pages, whose members are every code
// sharing a 2-character prefix plus opportunities classified on the group itself.
func (db *DB) GetSEOPageOpportunitiesByPrefix(ctx context.Context, filterCol, prefix string, limit int) ([]SEOOpportunity, error) {
	allowed := map[string]bool{"classification_code": true}
	if !allowed[filterCol] {
		return nil, fmt.Errorf("invalid filter column: %s", filterCol)
	}

	query := fmt.Sprintf(`
		SELECT notice_id, COALESCE(title, '') AS title, COALESCE(department, '') AS department,
		       naics_code, set_aside_code,
		       posted_date, response_deadline, pop_state, solicitation_number
		FROM opportunities
		WHERE %s LIKE $1 AND active = true AND is_latest = true
		ORDER BY posted_date DESC LIMIT $2
	`, filterCol)

	rows, err := db.pool.Query(ctx, query, prefix+"%", limit)
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
