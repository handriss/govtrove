package repository

import (
	"context"
	"time"
)

type CodeLookupStats struct {
	Total       int     `json:"total"`
	Naics       int     `json:"naics"`
	Psc         int     `json:"psc"`
	ZeroResults int     `json:"zero_results"`
	Last30d     int     `json:"last_30d"`
	Last7d      int     `json:"last_7d"`
	AvgTopScore float64 `json:"avg_top_score"`
}

type CodeLookupAgg struct {
	Description string `json:"description"`
	CodeType   string `json:"code_type"`
	Count      int    `json:"count"`
}

type CodeLookupRow struct {
	ID          int64     `json:"id"`
	CodeType    string    `json:"code_type"`
	Description string    `json:"description"`
	ResultCount int       `json:"result_count"`
	TopCode     *string   `json:"top_code"`
	TopScore    *float64  `json:"top_score"`
	CreatedAt   time.Time `json:"created_at"`
}

type CodeLookupAnalytics struct {
	Stats      CodeLookupStats `json:"stats"`
	Popular    []CodeLookupAgg `json:"popular"`
	ZeroResult []CodeLookupAgg `json:"zero_result"`
	Recent     []CodeLookupRow `json:"recent"`
}

// CodeLookupAnalytics summarizes NAICS/PSC code-finder usage from the code_lookups log.
func (r *PipelineRepository) CodeLookupAnalytics(ctx context.Context, recentLimit int) (*CodeLookupAnalytics, error) {
	out := &CodeLookupAnalytics{
		Popular:    []CodeLookupAgg{},
		ZeroResult: []CodeLookupAgg{},
		Recent:     []CodeLookupRow{},
	}

	err := r.pool.QueryRow(ctx, `
		SELECT
			COUNT(*),
			COUNT(*) FILTER (WHERE code_type = 'naics'),
			COUNT(*) FILTER (WHERE code_type = 'psc'),
			COUNT(*) FILTER (WHERE result_count = 0),
			COUNT(*) FILTER (WHERE created_at > now() - interval '30 days'),
			COUNT(*) FILTER (WHERE created_at > now() - interval '7 days'),
			COALESCE(AVG(top_score) FILTER (WHERE top_score IS NOT NULL), 0)
		FROM code_lookups
	`).Scan(&out.Stats.Total, &out.Stats.Naics, &out.Stats.Psc, &out.Stats.ZeroResults,
		&out.Stats.Last30d, &out.Stats.Last7d, &out.Stats.AvgTopScore)
	if err != nil {
		return nil, err
	}

	aggQuery := func(where string) ([]CodeLookupAgg, error) {
		rows, err := r.pool.Query(ctx, `
			SELECT lower(btrim(description)) AS d, code_type, COUNT(*) AS n
			FROM code_lookups `+where+`
			GROUP BY d, code_type
			ORDER BY n DESC
			LIMIT 25`)
		if err != nil {
			return nil, err
		}
		defer rows.Close()
		aggs := []CodeLookupAgg{}
		for rows.Next() {
			var a CodeLookupAgg
			if err := rows.Scan(&a.Description, &a.CodeType, &a.Count); err != nil {
				return nil, err
			}
			aggs = append(aggs, a)
		}
		return aggs, rows.Err()
	}

	if out.Popular, err = aggQuery(""); err != nil {
		return nil, err
	}
	if out.ZeroResult, err = aggQuery("WHERE result_count = 0"); err != nil {
		return nil, err
	}

	if recentLimit <= 0 || recentLimit > 200 {
		recentLimit = 100
	}
	recRows, err := r.pool.Query(ctx, `
		SELECT id, code_type, description, result_count, top_code, top_score, created_at
		FROM code_lookups
		ORDER BY created_at DESC
		LIMIT $1`, recentLimit)
	if err != nil {
		return nil, err
	}
	defer recRows.Close()
	for recRows.Next() {
		var row CodeLookupRow
		if err := recRows.Scan(&row.ID, &row.CodeType, &row.Description, &row.ResultCount, &row.TopCode, &row.TopScore, &row.CreatedAt); err != nil {
			return nil, err
		}
		out.Recent = append(out.Recent, row)
	}
	return out, recRows.Err()
}
