package events

import (
	"context"
	"fmt"
	"math/big"
	"sync"

	"github.com/chain4travel/camino-messenger-contracts/go/contracts/bookingtoken"
	"github.com/chain4travel/camino-messenger-contracts/go/contracts/cmaccount"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/ethereum/go-ethereum/event"
	"github.com/google/uuid"
	"go.uber.org/zap"
)

// EventHandler defines the type for custom event handlers
type EventHandler func(event interface{})

// ListenerHandle is a handle returned when registering an event handler,
// which can be used to stop listening to that event
type ListenerHandle interface {
	Stop()
}

// EventListener listens for events from CMAccount and BookingToken contracts
type EventListener struct {
	client        *ethclient.Client
	logger        *zap.SugaredLogger
	mu            sync.Mutex
	cmAccounts    map[common.Address]*cmaccount.Cmaccount
	btContracts   map[common.Address]*bookingtoken.Bookingtoken
	subscriptions map[string]event.Subscription // Keyed by unique IDs
}

// NewEventListener creates a new EventListener instance
func NewEventListener(client *ethclient.Client, logger *zap.SugaredLogger) *EventListener {
	return &EventListener{
		client:        client,
		logger:        logger,
		cmAccounts:    make(map[common.Address]*cmaccount.Cmaccount),
		btContracts:   make(map[common.Address]*bookingtoken.Bookingtoken),
		subscriptions: make(map[string]event.Subscription),
	}
}

// RegisterServiceAddedHandler registers a handler for the ServiceAdded event on a CMAccount
func (el *EventListener) RegisterServiceAddedHandler(cmAccountAddr common.Address, serviceName []string, handler EventHandler) (ListenerHandle, error) {
	el.mu.Lock()
	defer el.mu.Unlock()

	// Get or create CMAccount instance
	cmAccount, err := el.getOrCreateCMAccount(cmAccountAddr)
	if err != nil {
		return nil, err
	}

	// Create channel for events
	eventChan := make(chan *cmaccount.CmaccountServiceAdded)
	opts := &bind.WatchOpts{Context: context.Background(), Start: nil}

	// Subscribe to ServiceAdded event
	sub, err := cmAccount.WatchServiceAdded(opts, eventChan, serviceName)
	if err != nil {
		return nil, fmt.Errorf("failed to subscribe to ServiceAdded events: %w", err)
	}

	// Generate a unique key for this subscription
	subID := uuid.New().String()
	el.subscriptions[subID] = sub

	// Start goroutine to listen for events
	go func() {
		for {
			select {
			case event := <-eventChan:
				handler(event)
			case err := <-sub.Err():
				el.logger.Errorf("Error in ServiceAdded subscription: %v", err)
				el.unsubscribe(subID)
				return
			}
		}
	}()

	// Return handle to stop listening
	return &listenerHandle{
		unsubscribe: func() {
			sub.Unsubscribe()
			el.unsubscribe(subID)
		},
	}, nil
}

// RegisterServiceFeeUpdatedHandler registers a handler for the ServiceFeeUpdated event on a CMAccount
func (el *EventListener) RegisterServiceFeeUpdatedHandler(cmAccountAddr common.Address, serviceName []string, handler EventHandler) (ListenerHandle, error) {
	el.mu.Lock()
	defer el.mu.Unlock()

	// Get or create CMAccount instance
	cmAccount, err := el.getOrCreateCMAccount(cmAccountAddr)
	if err != nil {
		return nil, err
	}

	// Create channel for events
	eventChan := make(chan *cmaccount.CmaccountServiceFeeUpdated)
	opts := &bind.WatchOpts{Context: context.Background(), Start: nil}

	// Subscribe to ServiceFeeUpdated event
	sub, err := cmAccount.WatchServiceFeeUpdated(opts, eventChan, serviceName)
	if err != nil {
		return nil, fmt.Errorf("failed to subscribe to ServiceFeeUpdated events: %w", err)
	}

	// Generate a unique key for this subscription
	subID := uuid.New().String()
	el.subscriptions[subID] = sub

	// Start goroutine to listen for events
	go func() {
		for {
			select {
			case event := <-eventChan:
				handler(event)
			case err := <-sub.Err():
				el.logger.Errorf("Error in ServiceFeeUpdated subscription: %v", err)
				el.unsubscribe(subID)
				return
			}
		}
	}()

	// Return handle to stop listening
	return &listenerHandle{
		unsubscribe: func() {
			sub.Unsubscribe()
			el.unsubscribe(subID)
		},
	}, nil
}

// RegisterTokenBoughtHandler registers a handler for TokenBought events on a BookingToken contract, filtered by tokenId and buyer
func (el *EventListener) RegisterTokenBoughtHandler(btAddress common.Address, tokenId []*big.Int, buyer []common.Address, handler EventHandler) (ListenerHandle, error) {
	el.mu.Lock()
	defer el.mu.Unlock()

	// Get or create BookingToken instance
	btContract, err := el.getOrCreateBookingToken(btAddress)
	if err != nil {
		return nil, err
	}

	// Create channel for events
	eventChan := make(chan *bookingtoken.BookingtokenTokenBought)
	opts := &bind.WatchOpts{Context: context.Background(), Start: nil}

	// Subscribe to TokenBought event with filters
	sub, err := btContract.WatchTokenBought(opts, eventChan, tokenId, buyer)
	if err != nil {
		return nil, fmt.Errorf("failed to subscribe to TokenBought events: %w", err)
	}

	// Generate a unique key for the subscription
	subID := uuid.New().String()
	el.subscriptions[subID] = sub

	// Start goroutine to listen for events
	go func() {
		for {
			select {
			case event := <-eventChan:
				handler(event)
			case err := <-sub.Err():
				el.logger.Errorf("Error in TokenBought subscription: %v", err)
				el.unsubscribe(subID)
				return
			}
		}
	}()

	// Return handle to stop listening
	return &listenerHandle{
		unsubscribe: func() {
			sub.Unsubscribe()
			el.unsubscribe(subID)
		},
	}, nil
}

// RegisterTokenReservedHandler registers a handler for TokenReserved events on a BookingToken contract, filtered by tokenId, reservedFor, and supplier
func (el *EventListener) RegisterTokenReservedHandler(btAddress common.Address, tokenId []*big.Int, reservedFor []common.Address, supplier []common.Address, handler EventHandler) (ListenerHandle, error) {
	el.mu.Lock()
	defer el.mu.Unlock()

	// Get or create BookingToken instance
	btContract, err := el.getOrCreateBookingToken(btAddress)
	if err != nil {
		return nil, err
	}

	// Create channel for events
	eventChan := make(chan *bookingtoken.BookingtokenTokenReserved)
	opts := &bind.WatchOpts{Context: context.Background(), Start: nil}

	// Subscribe to TokenReserved event with filters
	sub, err := btContract.WatchTokenReserved(opts, eventChan, tokenId, reservedFor, supplier)
	if err != nil {
		return nil, fmt.Errorf("failed to subscribe to TokenReserved events: %w", err)
	}

	// Generate a unique key for the subscription
	subID := uuid.New().String()
	el.subscriptions[subID] = sub

	// Start goroutine to listen for events
	go func() {
		for {
			select {
			case event := <-eventChan:
				handler(event)
			case err := <-sub.Err():
				el.logger.Errorf("Error in TokenReserved subscription: %v", err)
				el.unsubscribe(subID)
				return
			}
		}
	}()

	// Return handle to stop listening
	return &listenerHandle{
		unsubscribe: func() {
			sub.Unsubscribe()
			el.unsubscribe(subID)
		},
	}, nil
}

// getOrCreateCMAccount gets or creates a CMAccount instance
func (el *EventListener) getOrCreateCMAccount(addr common.Address) (*cmaccount.Cmaccount, error) {
	if cm, exists := el.cmAccounts[addr]; exists {
		return cm, nil
	}
	cm, err := cmaccount.NewCmaccount(addr, el.client)
	if err != nil {
		return nil, fmt.Errorf("failed to create CMAccount instance: %w", err)
	}
	el.cmAccounts[addr] = cm
	return cm, nil
}

// getOrCreateBookingToken gets or creates a BookingToken instance
func (el *EventListener) getOrCreateBookingToken(addr common.Address) (*bookingtoken.Bookingtoken, error) {
	if bt, exists := el.btContracts[addr]; exists {
		return bt, nil
	}
	bt, err := bookingtoken.NewBookingtoken(addr, el.client)
	if err != nil {
		return nil, fmt.Errorf("failed to create BookingToken instance: %w", err)
	}
	el.btContracts[addr] = bt
	return bt, nil
}

// unsubscribe removes a subscription from the subscriptions map
func (el *EventListener) unsubscribe(subID string) {
	el.mu.Lock()
	defer el.mu.Unlock()
	if sub, exists := el.subscriptions[subID]; exists {
		sub.Unsubscribe()
		delete(el.subscriptions, subID)
	}
}

// listenerHandle is a handle to stop event listeners
type listenerHandle struct {
	unsubscribe func()
}

// Stop stops the event listener
func (h *listenerHandle) Stop() {
	h.unsubscribe()
}
