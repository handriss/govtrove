package repository

import (
	"context"
	"fmt"
	"strings"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/handriss/govtrove/api/internal/models"
)

type CodeRepository struct {
	pool *pgxpool.Pool
}

func NewCodeRepository(pool *pgxpool.Pool) *CodeRepository {
	return &CodeRepository{pool: pool}
}

// LogLookup records one code-finder request for usage analytics. topCode/topScore
// are stored NULL when a lookup returns no matches.
func (r *CodeRepository) LogLookup(ctx context.Context, codeType, description string, resultCount int, topCode string, topScore float64) error {
	var tc *string
	var ts *float64
	if topCode != "" {
		tc = &topCode
		ts = &topScore
	}
	_, err := r.pool.Exec(ctx,
		`INSERT INTO code_lookups (code_type, description, result_count, top_code, top_score)
		 VALUES ($1, $2, $3, $4, $5)`,
		codeType, description, resultCount, tc, ts)
	if err != nil {
		return fmt.Errorf("logging code lookup: %w", err)
	}
	return nil
}

func (r *CodeRepository) FindSimilar(ctx context.Context, codeType string, embedding []float32, limit int) ([]models.CodeMatch, error) {
	vec := formatVector(embedding)
	query := `
		SELECT code, title, level, 1 - (embedding <=> $1::vector) AS similarity
		FROM code_embeddings
		WHERE code_type = $2
		ORDER BY embedding <=> $1::vector
		LIMIT $3
	`
	rows, err := r.pool.Query(ctx, query, vec, codeType, limit)
	if err != nil {
		return nil, fmt.Errorf("finding similar codes: %w", err)
	}
	defer rows.Close()

	var matches []models.CodeMatch
	for rows.Next() {
		var m models.CodeMatch
		if err := rows.Scan(&m.Code, &m.Title, &m.Level, &m.Similarity); err != nil {
			return nil, fmt.Errorf("scanning code match: %w", err)
		}
		matches = append(matches, m)
	}
	return matches, rows.Err()
}

func (r *CodeRepository) GetCorrelatedPSC(ctx context.Context, naicsCodes []string, limit int) ([]models.CodeCorrelation, error) {
	if len(naicsCodes) == 0 {
		return nil, nil
	}
	placeholders := make([]string, len(naicsCodes))
	args := make([]any, len(naicsCodes))
	for i, code := range naicsCodes {
		placeholders[i] = fmt.Sprintf("$%d", i+1)
		args[i] = code
	}
	args = append(args, limit)

	query := fmt.Sprintf(`
		SELECT c.psc_code, COALESCE(e.title, c.psc_code) AS title, SUM(c.co_occurrence_count)::int AS total
		FROM code_correlations c
		LEFT JOIN code_embeddings e ON e.code_type = 'psc' AND e.code = c.psc_code
		WHERE c.naics_code IN (%s)
		GROUP BY c.psc_code, e.title
		ORDER BY total DESC
		LIMIT $%d
	`, strings.Join(placeholders, ","), len(naicsCodes)+1)

	rows, err := r.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("getting correlated PSC codes: %w", err)
	}
	defer rows.Close()

	var correlations []models.CodeCorrelation
	for rows.Next() {
		var c models.CodeCorrelation
		if err := rows.Scan(&c.Code, &c.Title, &c.CoOccurrenceCount); err != nil {
			return nil, fmt.Errorf("scanning correlation: %w", err)
		}
		correlations = append(correlations, c)
	}
	return correlations, rows.Err()
}

func (r *CodeRepository) GetCorrelatedNAICS(ctx context.Context, pscCodes []string, limit int) ([]models.CodeCorrelation, error) {
	if len(pscCodes) == 0 {
		return nil, nil
	}
	placeholders := make([]string, len(pscCodes))
	args := make([]any, len(pscCodes))
	for i, code := range pscCodes {
		placeholders[i] = fmt.Sprintf("$%d", i+1)
		args[i] = code
	}
	args = append(args, limit)

	query := fmt.Sprintf(`
		SELECT c.naics_code, COALESCE(e.title, c.naics_code) AS title, SUM(c.co_occurrence_count)::int AS total
		FROM code_correlations c
		LEFT JOIN code_embeddings e ON e.code_type = 'naics' AND e.code = c.naics_code
		WHERE c.psc_code IN (%s)
		GROUP BY c.naics_code, e.title
		ORDER BY total DESC
		LIMIT $%d
	`, strings.Join(placeholders, ","), len(pscCodes)+1)

	rows, err := r.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("getting correlated NAICS codes: %w", err)
	}
	defer rows.Close()

	var correlations []models.CodeCorrelation
	for rows.Next() {
		var c models.CodeCorrelation
		if err := rows.Scan(&c.Code, &c.Title, &c.CoOccurrenceCount); err != nil {
			return nil, fmt.Errorf("scanning correlation: %w", err)
		}
		correlations = append(correlations, c)
	}
	return correlations, rows.Err()
}

func (r *CodeRepository) GetCorrelations(ctx context.Context, codeType string, codes []string, limit int) ([]models.CodeCorrelation, error) {
	if codeType == "naics" {
		return r.GetCorrelatedPSC(ctx, codes, limit)
	}
	return r.GetCorrelatedNAICS(ctx, codes, limit)
}

func formatVector(v []float32) string {
	parts := make([]string, len(v))
	for i, f := range v {
		parts[i] = fmt.Sprintf("%g", f)
	}
	return "[" + strings.Join(parts, ",") + "]"
}
