package cache

import (
	"sync"
	"time"

	accommodationv1 "buf.build/gen/go/chain4travel/camino-messenger-protocol/protocolbuffers/go/cmp/services/accommodation/v1"
	accommodationv2 "buf.build/gen/go/chain4travel/camino-messenger-protocol/protocolbuffers/go/cmp/services/accommodation/v2"
)

type searchCache struct {
	mu       sync.RWMutex
	cache_v1 map[string]*cachedResultV1
	cache_v2 map[string]*cachedResultV2
}

type cachedResultV1 struct {
	results []*accommodationv1.AccommodationSearchResult
	expiry  time.Time
}

type cachedResultV2 struct {
	results []*accommodationv2.AccommodationSearchResult
	expiry  time.Time
}

var Cache = &searchCache{
	cache_v1: make(map[string]*cachedResultV1),
	cache_v2: make(map[string]*cachedResultV2),
}

func (c *searchCache) SetV1(key string, results []*accommodationv1.AccommodationSearchResult) {
	c.mu.Lock()
	defer c.mu.Unlock()

	// Clean up expired entries first
	now := time.Now()
	for k, item := range c.cache_v1 {
		if now.After(item.expiry) {
			delete(c.cache_v1, k)
		}
	}

	// Add new entry
	c.cache_v1[key] = &cachedResultV1{
		results: results,
		expiry:  now.Add(1 * time.Hour),
	}
}

func (c *searchCache) GetV1(key string) ([]*accommodationv1.AccommodationSearchResult, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	if cached, ok := c.cache_v1[key]; ok && time.Now().Before(cached.expiry) {
		return cached.results, true
	}
	return nil, false
}

func (c *searchCache) SetV2(key string, results []*accommodationv2.AccommodationSearchResult) {
	c.mu.Lock()
	defer c.mu.Unlock()

	// Clean up expired entries first
	now := time.Now()
	for k, item := range c.cache_v2 {
		if now.After(item.expiry) {
			delete(c.cache_v2, k)
		}
	}

	// Add new entry
	c.cache_v2[key] = &cachedResultV2{
		results: results,
		expiry:  now.Add(1 * time.Hour),
	}
}

func (c *searchCache) GetV2(key string) ([]*accommodationv2.AccommodationSearchResult, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	if cached, ok := c.cache_v2[key]; ok && time.Now().Before(cached.expiry) {
		return cached.results, true
	}
	return nil, false
}
