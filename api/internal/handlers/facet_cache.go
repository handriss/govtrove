package handlers

import (
	"container/list"
	"context"
	"sync"
	"time"

	"golang.org/x/sync/singleflight"

	"github.com/handriss/govtrove/api/internal/models"
)

// The unfiltered facet query scans every row and was the largest single
// consumer of Neon compute; counts only move when the pipeline ingests, once a day.
// On a cold Neon compute the six sequential facet statements take 50-90s, past
// the 60s edge timeout, so a visitor must never be the one waiting on a reload:
// expired entries keep being served while a background load replaces them.
const (
	facetFreshFor          = 6 * time.Hour
	facetServeStaleFor     = 24 * time.Hour
	facetCacheMaxEntries   = 256
	facetForegroundTimeout = 60 * time.Second
	facetBackgroundTimeout = 3 * time.Minute
	facetRetryRefreshAfter = 10 * time.Minute
)

type facetCacheEntry struct {
	key        string
	result     *models.FacetResult
	freshUntil time.Time
	staleUntil time.Time
}

// Least-recently-used eviction: crawlers send thousands of one-off queries, and
// the old wipe-everything-when-full policy threw away the hot default-view entry
// along with them, so every visitor paid the full scan again.
type facetCache struct {
	mu             sync.Mutex
	entries        map[string]*list.Element
	recency        *list.List
	group          singleflight.Group
	now            func() time.Time
	onRefreshError func(key string, err error)
}

func newFacetCache() *facetCache {
	return &facetCache{
		entries:        make(map[string]*list.Element),
		recency:        list.New(),
		now:            time.Now,
		onRefreshError: func(string, error) {},
	}
}

func (c *facetCache) get(ctx context.Context, key string, load func(context.Context) (*models.FacetResult, error)) (*models.FacetResult, error) {
	if entry := c.lookup(key); entry != nil {
		if c.now().After(entry.freshUntil) {
			c.refreshInBackground(key, load)
		}
		return entry.result, nil
	}

	v, err, _ := c.group.Do(key, func() (interface{}, error) {
		// Detached so one caller disconnecting doesn't fail everyone waiting on the same flight.
		return c.loadAndStore(context.WithoutCancel(ctx), key, load, facetForegroundTimeout)
	})
	if err != nil {
		return nil, err
	}
	return v.(*models.FacetResult), nil
}

func (c *facetCache) refreshInBackground(key string, load func(context.Context) (*models.FacetResult, error)) {
	c.group.DoChan(key, func() (interface{}, error) {
		result, err := c.loadAndStore(context.Background(), key, load, facetBackgroundTimeout)
		if err != nil {
			c.postponeRefresh(key)
			c.onRefreshError(key, err)
		}
		return result, err
	})
}

func (c *facetCache) loadAndStore(ctx context.Context, key string, load func(context.Context) (*models.FacetResult, error), timeout time.Duration) (*models.FacetResult, error) {
	loadCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()
	result, err := load(loadCtx)
	if err != nil {
		return nil, err
	}
	c.store(key, result)
	return result, nil
}

func (c *facetCache) postponeRefresh(key string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if el, ok := c.entries[key]; ok {
		el.Value.(*facetCacheEntry).freshUntil = c.now().Add(facetRetryRefreshAfter)
	}
}

func (c *facetCache) lookup(key string) *facetCacheEntry {
	c.mu.Lock()
	defer c.mu.Unlock()
	el, ok := c.entries[key]
	if !ok {
		return nil
	}
	entry := el.Value.(*facetCacheEntry)
	if c.now().After(entry.staleUntil) {
		return nil
	}
	c.recency.MoveToFront(el)
	return entry
}

func (c *facetCache) store(key string, result *models.FacetResult) {
	now := c.now()
	entry := &facetCacheEntry{key: key, result: result, freshUntil: now.Add(facetFreshFor), staleUntil: now.Add(facetServeStaleFor)}
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
