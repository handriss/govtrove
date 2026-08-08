package searchrescue

import (
	"sync"
	"time"
)

type cache struct {
	mu      sync.Mutex
	entries map[string]cacheEntry
	ttl     time.Duration
	max     int
}

type cacheEntry struct {
	res *Result
	exp time.Time
}

func newCache(ttl time.Duration, max int) *cache {
	return &cache{entries: make(map[string]cacheEntry), ttl: ttl, max: max}
}

func (c *cache) get(key string) (*Result, bool) {
	c.mu.Lock()
	defer c.mu.Unlock()
	e, ok := c.entries[key]
	if !ok || time.Now().After(e.exp) {
		return nil, false
	}
	return e.res, true
}

func (c *cache) set(key string, res *Result) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if len(c.entries) >= c.max {
		now := time.Now()
		for k, e := range c.entries {
			if now.After(e.exp) {
				delete(c.entries, k)
			}
		}
		for k := range c.entries {
			if len(c.entries) < c.max {
				break
			}
			delete(c.entries, k)
		}
	}
	c.entries[key] = cacheEntry{res: res, exp: time.Now().Add(c.ttl)}
}
