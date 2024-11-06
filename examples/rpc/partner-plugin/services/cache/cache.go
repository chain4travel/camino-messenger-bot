package cache

import (
	"sync"
	"time"

	accommodationv1 "buf.build/gen/go/chain4travel/camino-messenger-protocol/protocolbuffers/go/cmp/services/accommodation/v1"
)

type searchCache struct {
	mu    sync.RWMutex
	cache map[string]*cachedResult
}

type cachedResult struct {
	results []*accommodationv1.AccommodationSearchResult
	expiry  time.Time
}

var Cache = &searchCache{
	cache: make(map[string]*cachedResult),
}

func (c *searchCache) Set(key string, results []*accommodationv1.AccommodationSearchResult) {
	c.mu.Lock()
	defer c.mu.Unlock()

	// Clean up expired entries first
	now := time.Now()
	for k, item := range c.cache {
		if now.After(item.expiry) {
			delete(c.cache, k)
		}
	}

	// Add new entry
	c.cache[key] = &cachedResult{
		results: results,
		expiry:  now.Add(1 * time.Hour),
	}
}

func (c *searchCache) Get(key string) ([]*accommodationv1.AccommodationSearchResult, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	if cached, ok := c.cache[key]; ok && time.Now().Before(cached.expiry) {
		return cached.results, true
	}
	return nil, false
}
