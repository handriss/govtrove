package repository

import (
	"context"
	"strings"
	"sync"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/handriss/govtrove/api/internal/models"
)

type GeoSynonymRepository struct {
	pool *pgxpool.Pool

	mu        sync.RWMutex
	cache     map[string][]models.GeoSynonym
	loadedAt  time.Time
	cacheTTL  time.Duration
}

func NewGeoSynonymRepository(pool *pgxpool.Pool) *GeoSynonymRepository {
	return &GeoSynonymRepository{
		pool:     pool,
		cacheTTL: 1 * time.Hour,
	}
}

func (r *GeoSynonymRepository) Lookup(ctx context.Context, query string) ([]models.GeoSynonym, error) {
	if err := r.ensureCache(ctx); err != nil {
		return nil, err
	}

	r.mu.RLock()
	defer r.mu.RUnlock()

	key := strings.ToLower(strings.TrimSpace(query))
	if matches, ok := r.cache[key]; ok {
		return matches, nil
	}
	return nil, nil
}

func (r *GeoSynonymRepository) ensureCache(ctx context.Context) error {
	r.mu.RLock()
	if r.cache != nil && time.Since(r.loadedAt) < r.cacheTTL {
		r.mu.RUnlock()
		return nil
	}
	r.mu.RUnlock()

	r.mu.Lock()
	defer r.mu.Unlock()

	// Double-check after acquiring write lock
	if r.cache != nil && time.Since(r.loadedAt) < r.cacheTTL {
		return nil
	}

	rows, err := r.pool.Query(ctx, `SELECT id, term, match_type, states, cities FROM geo_synonyms`)
	if err != nil {
		return err
	}
	defer rows.Close()

	cache := make(map[string][]models.GeoSynonym)
	for rows.Next() {
		var s models.GeoSynonym
		if err := rows.Scan(&s.ID, &s.Term, &s.MatchType, &s.States, &s.Cities); err != nil {
			return err
		}
		key := strings.ToLower(s.Term)
		cache[key] = append(cache[key], s)
	}

	r.cache = cache
	r.loadedAt = time.Now()
	return nil
}
