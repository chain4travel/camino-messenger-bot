package cache

import (
	"sync"
	"time"

	accommodationv1 "buf.build/gen/go/chain4travel/camino-messenger-protocol/protocolbuffers/go/cmp/services/accommodation/v1"
	accommodationv2 "buf.build/gen/go/chain4travel/camino-messenger-protocol/protocolbuffers/go/cmp/services/accommodation/v2"
	typesv1 "buf.build/gen/go/chain4travel/camino-messenger-protocol/protocolbuffers/go/cmp/types/v1"
	typesv2 "buf.build/gen/go/chain4travel/camino-messenger-protocol/protocolbuffers/go/cmp/types/v2"
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

type validationCache struct {
	mu       sync.RWMutex
	cache_v1 map[string]*cachedValidationV1
	cache_v2 map[string]*cachedValidationV2
}

type cachedValidationV1 struct {
	priceDetail *typesv1.PriceDetail
	expiry      time.Time
}

type cachedValidationV2 struct {
	priceDetail *typesv2.PriceDetail
	expiry      time.Time
}

// Constructor for searchCache
var (
	searchOnce         sync.Once
	validationOnce     sync.Once
	searchInstance     *searchCache
	validationInstance *validationCache
)

func NewSearchCache() *searchCache {
	searchOnce.Do(func() {
		searchInstance = &searchCache{
			cache_v1: make(map[string]*cachedResultV1),
			cache_v2: make(map[string]*cachedResultV2),
		}
	})
	return searchInstance
}

// Constructor for validationCache
func NewValidationCache() *validationCache {
	validationOnce.Do(func() {
		validationInstance = &validationCache{
			cache_v1: make(map[string]*cachedValidationV1),
			cache_v2: make(map[string]*cachedValidationV2),
		}
	})
	return validationInstance
}

// SetV1 adds a new V1 search result to the cache
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

// GetV1 retrieves V1 search results from the cache
func (c *searchCache) GetV1(key string) ([]*accommodationv1.AccommodationSearchResult, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	if cached, ok := c.cache_v1[key]; ok && time.Now().Before(cached.expiry) {
		return cached.results, true
	}
	return nil, false
}

// SetV2 adds a new V2 search result to the cache
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

// GetV2 retrieves V2 search results from the cache
func (c *searchCache) GetV2(key string) ([]*accommodationv2.AccommodationSearchResult, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	if cached, ok := c.cache_v2[key]; ok && time.Now().Before(cached.expiry) {
		return cached.results, true
	}
	return nil, false
}

// Set adds a new priceDetail to the validation cache with the given validationId
func (vc *validationCache) SetV1(validationId string, priceDetail *typesv1.PriceDetail) {
	vc.mu.Lock()
	defer vc.mu.Unlock()

	// Clean up expired entries first
	now := time.Now()
	for k, item := range vc.cache_v1 {
		if now.After(item.expiry) {
			delete(vc.cache_v1, k)
		}
	}

	// Add new entry with 1-hour expiry
	vc.cache_v1[validationId] = &cachedValidationV1{
		priceDetail: priceDetail,
		expiry:      now.Add(1 * time.Hour),
	}
}

// Get retrieves the priceDetail associated with the given validationId
func (vc *validationCache) GetV1(validationId string) (*typesv1.PriceDetail, bool) {
	vc.mu.RLock()
	defer vc.mu.RUnlock()

	if cached, ok := vc.cache_v1[validationId]; ok && time.Now().Before(cached.expiry) {
		return cached.priceDetail, true
	}
	return nil, false
}

// Set adds a new priceDetail to the validation cache with the given validationId
func (vc *validationCache) SetV2(validationId string, priceDetail *typesv2.PriceDetail) {
	vc.mu.Lock()
	defer vc.mu.Unlock()

	// Clean up expired entries first
	now := time.Now()
	for k, item := range vc.cache_v2 {
		if now.After(item.expiry) {
			delete(vc.cache_v2, k)
		}
	}

	// Add new entry with 1-hour expiry
	vc.cache_v2[validationId] = &cachedValidationV2{
		priceDetail: priceDetail,
		expiry:      now.Add(1 * time.Hour),
	}
}

// Get retrieves the priceDetail associated with the given validationId
func (vc *validationCache) GetV2(validationId string) (*typesv2.PriceDetail, bool) {
	vc.mu.RLock()
	defer vc.mu.RUnlock()

	if cached, ok := vc.cache_v2[validationId]; ok && time.Now().Before(cached.expiry) {
		return cached.priceDetail, true
	}
	return nil, false
}
