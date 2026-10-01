package handlers

import (
	"context"
	"sync"
	"time"

	"golang.org/x/sync/singleflight"

	"github.com/handriss/govtrove/api/internal/models"
)

// The unfiltered facet query scans every row (~9s) and was the largest single
// consumer of Neon compute; counts only move when the pipeline ingests.
const (
	facetCacheTTL        = 10 * time.Minute
	facetCacheMaxEntries = 256
	facetQueryTimeout    = 60 * time.Second
)

type facetCacheEntry struct {
	result    *models.FacetResult
	expiresAt time.Time
}

type facetCache struct {
	mu      sync.Mutex
	entries map[string]facetCacheEntry
	group   singleflight.Group
}

func newFacetCache() *facetCache {
	return &facetCache{entries: make(map[string]facetCacheEntry)}
}

func (c *facetCache) get(ctx context.Context, key string, load func(context.Context) (*models.FacetResult, error)) (*models.FacetResult, error) {
	now := time.Now()
	c.mu.Lock()
	if e, ok := c.entries[key]; ok && now.Before(e.expiresAt) {
		c.mu.Unlock()
		return e.result, nil
	}
	c.mu.Unlock()

	v, err, _ := c.group.Do(key, func() (interface{}, error) {
		// Detached so one caller disconnecting doesn't fail everyone waiting on the same flight.
		loadCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), facetQueryTimeout)
		defer cancel()
		result, err := load(loadCtx)
		if err != nil {
			return nil, err
		}
		c.store(key, result)
		return result, nil
	})
	if err != nil {
		return nil, err
	}
	return v.(*models.FacetResult), nil
}

func (c *facetCache) store(key string, result *models.FacetResult) {
	now := time.Now()
	c.mu.Lock()
	defer c.mu.Unlock()
	if len(c.entries) >= facetCacheMaxEntries {
		for k, e := range c.entries {
			if !now.Before(e.expiresAt) {
				delete(c.entries, k)
			}
		}
		if len(c.entries) >= facetCacheMaxEntries {
			c.entries = make(map[string]facetCacheEntry)
		}
	}
	c.entries[key] = facetCacheEntry{result: result, expiresAt: now.Add(facetCacheTTL)}
}
