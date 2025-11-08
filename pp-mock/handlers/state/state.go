// Copyright (C) 2022-2025, Chain4Travel AG. All rights reserved.
// See the file LICENSE for licensing terms.

package state

import (
	"log"
	"sync"
	"time"
)

// Constants for entry timeout
// Must be bigger than config.BuyableUntilDefault
const EntryTimeout = 10 * time.Minute

type UnifiedPrice struct {
	Price                string
	Decimals             uint32
	IsNative             bool
	IsoCurrencyEnum      int32
	TokenContractAddress string
}

// SearchData represents the data for a search result and everything which is
// needed for this mock to properly represent a full workflow.
// This includes also data which is needed for the validate request and response.
// Such as information about seat selection and prices.
type SearchData struct {
	NumResults   int
	NumTravelers int
	SeatMapID    string
	Prices       []*UnifiedPrice // Validation price for the search results
	JSONRequest  string          // Mainly for debugging purposes
	JSONResponse string          // Mainly for debugging purposes
}

// ValidationData represents the data for a validation result.
type ValidationData struct {
	InitialSearchData SearchData
	VerifiedPrice     *UnifiedPrice
	JSONRequest       string // Mainly for debugging purposes
	JSONResponse      string // Mainly for debugging purposes
}

// SearchResult represents a unified cut down search result.
type SearchResult struct {
	Data      SearchData
	CreatedAt time.Time
}

// ValidationResult represents the validation result.
type ValidationResult struct {
	Data      ValidationData
	CreatedAt time.Time
}

type MintResult struct {
	MintID    string
	SeatMapID string
	CreatedAt time.Time
	Bought    bool
}

// Store holds the in-memory state.
type Store struct {
	searchResults     map[string]SearchResult
	validationResults map[string]ValidationResult
	mintResults       map[string]MintResult
	mu                sync.RWMutex
}

var (
	instance *Store
	once     sync.Once
)

// GetStore returns the singleton instance of Store.
// And starts the cleanup thread for expired entries.
func GetStore() *Store {
	once.Do(func() {
		instance = &Store{
			searchResults:     make(map[string]SearchResult),
			validationResults: make(map[string]ValidationResult),
			mintResults:       make(map[string]MintResult),
		}
		go instance.cleanupExpiredEntries()
	})
	return instance
}

// AddSearchResult adds a search result to the store.
func (s *Store) AddSearchResult(searchID string, data SearchData) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.searchResults[searchID] = SearchResult{
		Data:      data,
		CreatedAt: time.Now(),
	}
}

// GetSearchResult retrieves a search result from the store.
func (s *Store) GetSearchResult(searchID string) (SearchResult, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	result, exists := s.searchResults[searchID]
	return result, exists
}

// AddValidationResult adds a validation result to the store.
func (s *Store) AddValidationResult(validationID string, data ValidationData) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.validationResults[validationID] = ValidationResult{
		Data:      data,
		CreatedAt: time.Now(),
	}
}

// GetValidationResult retrieves a validation result from the store.
func (s *Store) GetValidationResult(validationID string) (ValidationResult, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	result, exists := s.validationResults[validationID]
	return result, exists
}

func (s *Store) AddMintResult(mintID string, seatMapID string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.mintResults[mintID] = MintResult{
		MintID:    mintID,
		SeatMapID: seatMapID,
		CreatedAt: time.Now(),
	}
}

func (s *Store) SetMintBought(mintID string, bought bool) {
	s.mu.Lock()
	defer s.mu.Unlock()

	result, exists := s.mintResults[mintID]
	if !exists {
		log.Printf("SetMintBought error: mint ID %s not found in store", mintID)
		return
	}

	result.Bought = bought
	s.mintResults[mintID] = result
}

func (s *Store) RemoveMintResult(mintID string) {
	s.mu.Lock()
	defer s.mu.Unlock()

	_, exists := s.mintResults[mintID]
	if !exists {
		log.Printf("RemoveMintResult error: mint ID %s not found in store", mintID)
		return
	}

	delete(s.mintResults, mintID)
}

func (s *Store) GetMintResult(mintID string) (MintResult, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	result, exists := s.mintResults[mintID]
	return result, exists
}

// cleanupExpiredEntries removes entries that have expired based on the entryTimeout.
func (s *Store) cleanupExpiredEntries() {
	for {
		time.Sleep(EntryTimeout)
		s.mu.Lock()
		now := time.Now()
		for id, result := range s.searchResults {
			if now.Sub(result.CreatedAt) > EntryTimeout {
				delete(s.searchResults, id)
			}
		}
		for id, result := range s.validationResults {
			if now.Sub(result.CreatedAt) > EntryTimeout {
				delete(s.validationResults, id)
			}
		}
		for id, result := range s.mintResults {
			if now.Sub(result.CreatedAt) > EntryTimeout {
				delete(s.mintResults, id)
			}
		}
		s.mu.Unlock()
	}
}
