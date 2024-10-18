package cache

import (
	"github.com/ethereum/go-ethereum/common"
	lru "github.com/hashicorp/golang-lru"
)

type TokenData struct {
	Contract *common.Address // Cached contract instance
	Decimals uint8           // Token decimals
}

type TokenCache struct {
	cache *lru.Cache
}

func NewTokenCache(size int) (*TokenCache, error) {
	c, err := lru.New(size)
	if err != nil {
		return nil, err
	}
	return &TokenCache{cache: c}, nil
}

func (tc *TokenCache) Get(contractAddress common.Address) (int32, bool) {
	value, ok := tc.cache.Get(contractAddress.Hex())
	if !ok {
		return 0, false
	}
	decimals, ok := value.(int32)
	if !ok {
		return 0, false
	}
	return decimals, true
}

func (tc *TokenCache) Add(contractAddress common.Address, decimals int32) {
	tc.cache.Add(contractAddress.Hex(), decimals)
}
