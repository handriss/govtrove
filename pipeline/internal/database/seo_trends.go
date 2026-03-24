package database

import (
	"context"
)

type NAICSVolume struct {
	NAICSCode string
	ThisWeek  int
	LastWeek  int
}

type AgencyVolume struct {
	Department string
	ThisWeek   int
	LastWeek   int
}

type TitleRow struct {
	Title string
}

type SetAsideCount struct {
	SetAsideCode        string
	SetAsideDescription string
	Count               int
}

func (db *DB) GetNAICSVolumeDelta(ctx context.Context) ([]NAICSVolume, error) {
	rows, err := db.pool.Query(ctx, `
		SELECT
			naics_code,
			COUNT(*) FILTER (WHERE posted_date >= NOW() - INTERVAL '7 days') AS this_week,
			COUNT(*) FILTER (WHERE posted_date >= NOW() - INTERVAL '14 days' AND posted_date < NOW() - INTERVAL '7 days') AS last_week
		FROM opportunities
		WHERE posted_date >= NOW() - INTERVAL '14 days'
		  AND naics_code IS NOT NULL
		  AND naics_code != ''
		  AND is_latest = true
		GROUP BY naics_code
		HAVING COUNT(*) FILTER (WHERE posted_date >= NOW() - INTERVAL '7 days') >= 5
		ORDER BY naics_code
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var results []NAICSVolume
	for rows.Next() {
		var r NAICSVolume
		if err := rows.Scan(&r.NAICSCode, &r.ThisWeek, &r.LastWeek); err != nil {
			return nil, err
		}
		results = append(results, r)
	}
	return results, rows.Err()
}

func (db *DB) GetAgencyVolumeDelta(ctx context.Context) ([]AgencyVolume, error) {
	rows, err := db.pool.Query(ctx, `
		SELECT
			department,
			COUNT(*) FILTER (WHERE posted_date >= NOW() - INTERVAL '7 days') AS this_week,
			COUNT(*) FILTER (WHERE posted_date >= NOW() - INTERVAL '14 days' AND posted_date < NOW() - INTERVAL '7 days') AS last_week
		FROM opportunities
		WHERE posted_date >= NOW() - INTERVAL '14 days'
		  AND department IS NOT NULL
		  AND department != ''
		  AND is_latest = true
		GROUP BY department
		HAVING COUNT(*) FILTER (WHERE posted_date >= NOW() - INTERVAL '7 days') >= 5
		ORDER BY department
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var results []AgencyVolume
	for rows.Next() {
		var r AgencyVolume
		if err := rows.Scan(&r.Department, &r.ThisWeek, &r.LastWeek); err != nil {
			return nil, err
		}
		results = append(results, r)
	}
	return results, rows.Err()
}

func (db *DB) GetRecentTitles(ctx context.Context) ([]TitleRow, error) {
	rows, err := db.pool.Query(ctx, `
		SELECT title
		FROM opportunities
		WHERE posted_date >= NOW() - INTERVAL '14 days'
		  AND title IS NOT NULL
		  AND title != ''
		  AND is_latest = true
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var results []TitleRow
	for rows.Next() {
		var r TitleRow
		if err := rows.Scan(&r.Title); err != nil {
			return nil, err
		}
		results = append(results, r)
	}
	return results, rows.Err()
}

func (db *DB) GetSetAsideCounts(ctx context.Context) ([]SetAsideCount, error) {
	rows, err := db.pool.Query(ctx, `
		SELECT set_aside_code, COALESCE(set_aside_description, ''), COUNT(*)
		FROM opportunities
		WHERE posted_date >= NOW() - INTERVAL '7 days'
		  AND set_aside_code IS NOT NULL
		  AND set_aside_code != ''
		  AND is_latest = true
		GROUP BY set_aside_code, set_aside_description
		ORDER BY COUNT(*) DESC
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var results []SetAsideCount
	for rows.Next() {
		var r SetAsideCount
		if err := rows.Scan(&r.SetAsideCode, &r.SetAsideDescription, &r.Count); err != nil {
			return nil, err
		}
		results = append(results, r)
	}
	return results, rows.Err()
}
