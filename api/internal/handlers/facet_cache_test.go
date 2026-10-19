package handlers

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/handriss/govtrove/api/internal/models"
)

func TestFacetCacheServesRepeatsFromMemory(t *testing.T) {
	c := newFacetCache()
	var loads int32
	load := func(context.Context) (*models.FacetResult, error) {
		atomic.AddInt32(&loads, 1)
		return &models.FacetResult{Total: 7}, nil
	}
	for i := 0; i < 3; i++ {
		got, err := c.get(context.Background(), "", load)
		if err != nil || got.Total != 7 {
			t.Fatalf("get = %v, %v", got, err)
		}
	}
	if loads != 1 {
		t.Fatalf("loads = %d, want 1", loads)
	}
}

func TestFacetCacheCollapsesConcurrentMisses(t *testing.T) {
	c := newFacetCache()
	var loads int32
	release := make(chan struct{})
	load := func(context.Context) (*models.FacetResult, error) {
		atomic.AddInt32(&loads, 1)
		<-release
		return &models.FacetResult{Total: 1}, nil
	}
	var wg sync.WaitGroup
	for i := 0; i < 5; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			c.get(context.Background(), "", load)
		}()
	}
	time.Sleep(50 * time.Millisecond)
	close(release)
	wg.Wait()
	if loads != 1 {
		t.Fatalf("loads = %d, want 1", loads)
	}
}

func TestFacetCacheDoesNotCacheErrors(t *testing.T) {
	c := newFacetCache()
	var loads int32
	load := func(context.Context) (*models.FacetResult, error) {
		if atomic.AddInt32(&loads, 1) == 1 {
			return nil, errors.New("db down")
		}
		return &models.FacetResult{Total: 2}, nil
	}
	if _, err := c.get(context.Background(), "k", load); err == nil {
		t.Fatal("want error on first load")
	}
	if got, err := c.get(context.Background(), "k", load); err != nil || got.Total != 2 {
		t.Fatalf("second get = %v, %v", got, err)
	}
}

func TestFacetCacheLoadSurvivesCallerCancellation(t *testing.T) {
	c := newFacetCache()
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	_, err := c.get(ctx, "", func(loadCtx context.Context) (*models.FacetResult, error) {
		return &models.FacetResult{}, loadCtx.Err()
	})
	if err != nil {
		t.Fatalf("load saw caller cancellation: %v", err)
	}
}

func TestFacetCacheKeepsHotEntryWhileOneOffQueriesOverflowIt(t *testing.T) {
	c := newFacetCache()
	var hotLoads int32
	hot := func(context.Context) (*models.FacetResult, error) {
		atomic.AddInt32(&hotLoads, 1)
		return &models.FacetResult{Total: 1}, nil
	}
	oneOff := func(context.Context) (*models.FacetResult, error) { return &models.FacetResult{}, nil }

	c.get(context.Background(), "default-view", hot)
	for i := 0; i < facetCacheMaxEntries*4; i++ {
		c.get(context.Background(), fmt.Sprintf("q=SOL%d", i), oneOff)
		if i%10 == 0 {
			c.get(context.Background(), "default-view", hot)
		}
	}
	if hotLoads != 1 {
		t.Fatalf("hot entry loaded %d times, want 1", hotLoads)
	}
	if len(c.entries) != facetCacheMaxEntries || c.recency.Len() != facetCacheMaxEntries {
		t.Fatalf("size = %d/%d, want %d", len(c.entries), c.recency.Len(), facetCacheMaxEntries)
	}
}

func TestFacetCacheEvictsLeastRecentlyUsedFirst(t *testing.T) {
	c := newFacetCache()
	load := func(context.Context) (*models.FacetResult, error) { return &models.FacetResult{}, nil }
	for i := 0; i <= facetCacheMaxEntries; i++ {
		c.get(context.Background(), fmt.Sprintf("k%d", i), load)
	}
	if _, ok := c.entries["k0"]; ok {
		t.Fatal("oldest entry k0 survived overflow")
	}
	if _, ok := c.entries["k1"]; !ok {
		t.Fatal("k1 evicted although only one slot was needed")
	}
}
