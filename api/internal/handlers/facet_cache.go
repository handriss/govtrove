package handlers

import (
	"container/list"
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
	key       string
	result    *models.FacetResult
	expiresAt time.Time
}

// Least-recently-used eviction: crawlers send thousands of one-off queries, and
// the old wipe-everything-when-full policy threw away the hot default-view entry
// along with them, so every visitor paid the full scan again.
type facetCache struct {
	mu      sync.Mutex
	entries map[string]*list.Element
	recency *list.List
	group   singleflight.Group
}

func newFacetCache() *facetCache {
	return &facetCache{entries: make(map[string]*list.Element), recency: list.New()}
}

func (c *facetCache) get(ctx context.Context, key string, load func(context.Context) (*models.FacetResult, error)) (*models.FacetResult, error) {
	now := time.Now()
	c.mu.Lock()
	if el, ok := c.entries[key]; ok {
		e := el.Value.(*facetCacheEntry)
		if now.Before(e.expiresAt) {
			c.recency.MoveToFront(el)
			c.mu.Unlock()
			return e.result, nil
		}
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
	entry := &facetCacheEntry{key: key, result: result, expiresAt: time.Now().Add(facetCacheTTL)}
	c.mu.Lock()
	defer c.mu.Unlock()
	if el, ok := c.entries[key]; ok {
		el.Value = entry
		c.recency.MoveToFront(el)
		return
	}
	c.entries[key] = c.recency.PushFront(entry)
	for c.recency.Len() > facetCacheMaxEntries {
		oldest := c.recency.Back()
		c.recency.Remove(oldest)
		delete(c.entries, oldest.Value.(*facetCacheEntry).key)
	}
}
