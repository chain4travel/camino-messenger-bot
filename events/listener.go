// listener.go

package events

import (
	"context"
	"fmt"
	"math/big"
	"sync"
	"time"

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
	Unsubscribe()
}

// subscriptionInfo holds information about a subscription
type subscriptionInfo struct {
	subID       string
	sub         event.Subscription
	contract    common.Address
	closeChan   func()
	handler     EventHandler
	unsubscribe context.CancelFunc // Used to cancel the resubscription
}

// EventListener listens for events from CMAccount and BookingToken contracts
type EventListener struct {
	client        *ethclient.Client
	logger        *zap.SugaredLogger
	mu            sync.Mutex
	cmAccounts    map[common.Address]*cmaccount.Cmaccount
	btContracts   map[common.Address]*bookingtoken.Bookingtoken
	subscriptions map[string]*subscriptionInfo // Keyed by unique IDs
}

// NewEventListener creates a new EventListener instance
func NewEventListener(client *ethclient.Client, logger *zap.SugaredLogger) *EventListener {
	return &EventListener{
		client:        client,
		logger:        logger,
		cmAccounts:    make(map[common.Address]*cmaccount.Cmaccount),
		btContracts:   make(map[common.Address]*bookingtoken.Bookingtoken),
		subscriptions: make(map[string]*subscriptionInfo),
	}
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

// unsubscribe removes a subscription from the subscriptions map and closes the event channel
func (el *EventListener) unsubscribe(subID string) {
	el.mu.Lock()
	defer el.mu.Unlock()
	if subInfo, exists := el.subscriptions[subID]; exists {
		// Cancel the context to stop resubscription
		subInfo.unsubscribe()
		// Unsubscribe from the event
		if subInfo.sub != nil {
			subInfo.sub.Unsubscribe()
		}
		// Close the event channel
		if subInfo.closeChan != nil {
			subInfo.closeChan()
		}
		delete(el.subscriptions, subID)
		el.logger.Debugf("Unsubscribed subID %s", subID)
	}
}

// listenerHandle is a handle to stop event listeners
type listenerHandle struct {
	unsubscribe func()
}

// Stop stops the event listener
func (h *listenerHandle) Unsubscribe() {
	h.unsubscribe()
}

// RegisterServiceAddedHandler registers a handler for the ServiceAdded event on a CMAccount
func (el *EventListener) RegisterServiceAddedHandler(cmAccountAddr common.Address, serviceName []string, handler EventHandler) (ListenerHandle, error) {
	el.mu.Lock()
	defer el.mu.Unlock()

	subID := uuid.New().String()

	// Get or create CMAccount instance
	cmAccount, err := el.getOrCreateCMAccount(cmAccountAddr)
	if err != nil {
		return nil, err
	}

	// Create a context with cancellation
	ctx, cancel := context.WithCancel(context.Background())

	// Store subscription info
	subInfo := &subscriptionInfo{
		subID:       subID,
		contract:    cmAccountAddr,
		handler:     handler,
		unsubscribe: cancel,
	}
	el.subscriptions[subID] = subInfo

	// Start the resubscription
	go el.resubscribeServiceAdded(ctx, subID, cmAccount, serviceName)

	// Return handle to stop listening
	return &listenerHandle{
		unsubscribe: func() {
			el.unsubscribe(subID)
		},
	}, nil
}

// resubscribeServiceAdded handles resubscription for ServiceAdded events
func (el *EventListener) resubscribeServiceAdded(ctx context.Context, subID string, cmAccount *cmaccount.Cmaccount, serviceName []string) {
	backoffMax := 2 * time.Minute // Maximum backoff time between retries

	resubscribeFn := func(ctx context.Context, lastError error) (event.Subscription, error) {
		if lastError != nil {
			el.logger.Errorf("Resubscribe attempt after error: %v", lastError)
		}

		eventChan := make(chan *cmaccount.CmaccountServiceAdded)

		// Subscribe to the event
		sub, err := cmAccount.WatchServiceAdded(&bind.WatchOpts{Context: ctx}, eventChan, serviceName)
		if err != nil {
			el.logger.Errorf("Failed to subscribe to ServiceAdded events: %v", err)
			return nil, err
		}

		el.mu.Lock()
		subInfo, exists := el.subscriptions[subID]
		if !exists {
			el.mu.Unlock()
			sub.Unsubscribe()
			return nil, fmt.Errorf("subscription %s no longer exists", subID)
		}
		subInfo.sub = sub
		subInfo.closeChan = func() {
			close(eventChan)
		}
		el.mu.Unlock()

		go el.listenForServiceAddedEvents(subID, eventChan)

		el.logger.Infof("Subscribed for ServiceAdded events on CMAccount %s", subInfo.contract)
		el.logger.Debugf("Subscription ID: %s", subID)
		el.logger.Debugf("Filters: %v", serviceName)

		return sub, nil
	}

	// Start resubscribing
	sub := event.ResubscribeErr(backoffMax, resubscribeFn)

	// Wait until context is canceled
	select {
	case <-ctx.Done():
		sub.Unsubscribe()
	}
}

// listenForServiceAddedEvents listens for ServiceAdded events
func (el *EventListener) listenForServiceAddedEvents(subID string, eventChan chan *cmaccount.CmaccountServiceAdded) {
	el.mu.Lock()
	subInfo, exists := el.subscriptions[subID]
	el.mu.Unlock()
	if !exists {
		return
	}
	handler := subInfo.handler

	for {
		select {
		case event, ok := <-eventChan:
			if !ok {
				// Channel closed, exit goroutine
				return
			}
			handler(event)
		}
	}
}

// RegisterServiceFeeUpdatedHandler registers a handler for the ServiceFeeUpdated event on a CMAccount
func (el *EventListener) RegisterServiceFeeUpdatedHandler(cmAccountAddr common.Address, serviceName []string, handler EventHandler) (ListenerHandle, error) {
	el.mu.Lock()
	defer el.mu.Unlock()

	subID := uuid.New().String()

	cmAccount, err := el.getOrCreateCMAccount(cmAccountAddr)
	if err != nil {
		return nil, err
	}

	ctx, cancel := context.WithCancel(context.Background())

	subInfo := &subscriptionInfo{
		subID:       subID,
		contract:    cmAccountAddr,
		handler:     handler,
		unsubscribe: cancel,
	}
	el.subscriptions[subID] = subInfo

	go el.resubscribeServiceFeeUpdated(ctx, subID, cmAccount, serviceName)

	return &listenerHandle{
		unsubscribe: func() {
			el.unsubscribe(subID)
		},
	}, nil
}

// resubscribeServiceFeeUpdated handles resubscription for ServiceFeeUpdated events
func (el *EventListener) resubscribeServiceFeeUpdated(ctx context.Context, subID string, cmAccount *cmaccount.Cmaccount, serviceName []string) {
	backoffMax := 2 * time.Minute

	resubscribeFn := func(ctx context.Context, lastError error) (event.Subscription, error) {
		if lastError != nil {
			el.logger.Errorf("Resubscribe attempt after error: %v", lastError)
		}

		eventChan := make(chan *cmaccount.CmaccountServiceFeeUpdated)

		sub, err := cmAccount.WatchServiceFeeUpdated(&bind.WatchOpts{Context: ctx}, eventChan, serviceName)
		if err != nil {
			el.logger.Errorf("Failed to subscribe to ServiceFeeUpdated events: %v", err)
			return nil, err
		}

		el.mu.Lock()
		subInfo, exists := el.subscriptions[subID]
		if !exists {
			el.mu.Unlock()
			sub.Unsubscribe()
			return nil, fmt.Errorf("subscription %s no longer exists", subID)
		}
		subInfo.sub = sub
		subInfo.closeChan = func() {
			close(eventChan)
		}
		el.mu.Unlock()

		go el.listenForServiceFeeUpdatedEvents(subID, eventChan)

		el.logger.Infof("Subscribed for ServiceFeeUpdated events on CMAccount %s", subInfo.contract)
		el.logger.Debugf("Subscription ID: %s", subID)
		el.logger.Debugf("Filters: %v", serviceName)

		return sub, nil
	}

	sub := event.ResubscribeErr(backoffMax, resubscribeFn)

	select {
	case <-ctx.Done():
		sub.Unsubscribe()
	}
}

// listenForServiceFeeUpdatedEvents listens for ServiceFeeUpdated events
func (el *EventListener) listenForServiceFeeUpdatedEvents(subID string, eventChan chan *cmaccount.CmaccountServiceFeeUpdated) {
	el.mu.Lock()
	subInfo, exists := el.subscriptions[subID]
	el.mu.Unlock()
	if !exists {
		return
	}
	handler := subInfo.handler

	for {
		select {
		case event, ok := <-eventChan:
			if !ok {
				return
			}
			handler(event)
		}
	}
}

// RegisterServiceRemovedHandler registers a handler for the ServiceRemoved event on a CMAccount
func (el *EventListener) RegisterServiceRemovedHandler(cmAccountAddr common.Address, serviceName []string, handler EventHandler) (ListenerHandle, error) {
	el.mu.Lock()
	defer el.mu.Unlock()

	subID := uuid.New().String()

	cmAccount, err := el.getOrCreateCMAccount(cmAccountAddr)
	if err != nil {
		return nil, err
	}

	ctx, cancel := context.WithCancel(context.Background())

	subInfo := &subscriptionInfo{
		subID:       subID,
		contract:    cmAccountAddr,
		handler:     handler,
		unsubscribe: cancel,
	}
	el.subscriptions[subID] = subInfo

	go el.resubscribeServiceRemoved(ctx, subID, cmAccount, serviceName)

	return &listenerHandle{
		unsubscribe: func() {
			el.unsubscribe(subID)
		},
	}, nil
}

// resubscribeServiceRemoved handles resubscription for ServiceRemoved events
func (el *EventListener) resubscribeServiceRemoved(ctx context.Context, subID string, cmAccount *cmaccount.Cmaccount, serviceName []string) {
	backoffMax := 2 * time.Minute

	resubscribeFn := func(ctx context.Context, lastError error) (event.Subscription, error) {
		if lastError != nil {
			el.logger.Errorf("Resubscribe attempt after error: %v", lastError)
		}

		eventChan := make(chan *cmaccount.CmaccountServiceRemoved)

		sub, err := cmAccount.WatchServiceRemoved(&bind.WatchOpts{Context: ctx}, eventChan, serviceName)
		if err != nil {
			el.logger.Errorf("Failed to subscribe to ServiceRemoved events: %v", err)
			return nil, err
		}

		el.mu.Lock()
		subInfo, exists := el.subscriptions[subID]
		if !exists {
			el.mu.Unlock()
			sub.Unsubscribe()
			return nil, fmt.Errorf("subscription %s no longer exists", subID)
		}
		subInfo.sub = sub
		subInfo.closeChan = func() {
			close(eventChan)
		}
		el.mu.Unlock()

		go el.listenForServiceRemovedEvents(subID, eventChan)

		el.logger.Infof("Subscribed for ServiceRemoved events on CMAccount %s", subInfo.contract)
		el.logger.Debugf("Subscription ID: %s", subID)
		el.logger.Debugf("Filters: %v", serviceName)

		return sub, nil
	}

	sub := event.ResubscribeErr(backoffMax, resubscribeFn)

	select {
	case <-ctx.Done():
		sub.Unsubscribe()
	}
}

// listenForServiceRemovedEvents listens for ServiceRemoved events
func (el *EventListener) listenForServiceRemovedEvents(subID string, eventChan chan *cmaccount.CmaccountServiceRemoved) {
	el.mu.Lock()
	subInfo, exists := el.subscriptions[subID]
	el.mu.Unlock()
	if !exists {
		return
	}
	handler := subInfo.handler

	for {
		select {
		case event, ok := <-eventChan:
			if !ok {
				return
			}
			handler(event)
		}
	}
}

// RegisterCMAccountUpgradedHandler registers a handler for the CMAccountUpgraded event on a CMAccount
func (el *EventListener) RegisterCMAccountUpgradedHandler(cmAccountAddr common.Address, oldImplementation []common.Address, newImplementation []common.Address, handler EventHandler) (ListenerHandle, error) {
	el.mu.Lock()
	defer el.mu.Unlock()

	subID := uuid.New().String()

	cmAccount, err := el.getOrCreateCMAccount(cmAccountAddr)
	if err != nil {
		return nil, err
	}

	ctx, cancel := context.WithCancel(context.Background())

	subInfo := &subscriptionInfo{
		subID:       subID,
		contract:    cmAccountAddr,
		handler:     handler,
		unsubscribe: cancel,
	}
	el.subscriptions[subID] = subInfo

	go el.resubscribeCMAccountUpgraded(ctx, subID, cmAccount, oldImplementation, newImplementation)

	return &listenerHandle{
		unsubscribe: func() {
			el.unsubscribe(subID)
		},
	}, nil
}

// resubscribeCMAccountUpgraded handles resubscription for CMAccountUpgraded events
func (el *EventListener) resubscribeCMAccountUpgraded(ctx context.Context, subID string, cmAccount *cmaccount.Cmaccount, oldImplementation []common.Address, newImplementation []common.Address) {
	backoffMax := 2 * time.Minute

	resubscribeFn := func(ctx context.Context, lastError error) (event.Subscription, error) {
		if lastError != nil {
			el.logger.Errorf("Resubscribe attempt after error: %v", lastError)
		}

		eventChan := make(chan *cmaccount.CmaccountCMAccountUpgraded)

		sub, err := cmAccount.WatchCMAccountUpgraded(&bind.WatchOpts{Context: ctx}, eventChan, oldImplementation, newImplementation)
		if err != nil {
			el.logger.Errorf("Failed to subscribe to CMAccountUpgraded events: %v", err)
			return nil, err
		}

		el.mu.Lock()
		subInfo, exists := el.subscriptions[subID]
		if !exists {
			el.mu.Unlock()
			sub.Unsubscribe()
			return nil, fmt.Errorf("subscription %s no longer exists", subID)
		}
		subInfo.sub = sub
		subInfo.closeChan = func() {
			close(eventChan)
		}
		el.mu.Unlock()

		go el.listenForCMAccountUpgradedEvents(subID, eventChan)

		el.logger.Infof("Subscribed for CMAccountUpgraded events on CMAccount %s", subInfo.contract)
		el.logger.Debugf("Subscription ID: %s", subID)
		el.logger.Debugf("Filters: oldImplementation: %v, newImplementation: %v", oldImplementation, newImplementation)

		return sub, nil
	}

	sub := event.ResubscribeErr(backoffMax, resubscribeFn)

	select {
	case <-ctx.Done():
		sub.Unsubscribe()
	}
}

// listenForCMAccountUpgradedEvents listens for CMAccountUpgraded events
func (el *EventListener) listenForCMAccountUpgradedEvents(subID string, eventChan chan *cmaccount.CmaccountCMAccountUpgraded) {
	el.mu.Lock()
	subInfo, exists := el.subscriptions[subID]
	el.mu.Unlock()
	if !exists {
		return
	}
	handler := subInfo.handler

	for {
		select {
		case event, ok := <-eventChan:
			if !ok {
				return
			}
			handler(event)
		}
	}
}

// RegisterTokenBoughtHandler registers a handler for TokenBought events on a BookingToken contract
func (el *EventListener) RegisterTokenBoughtHandler(btAddress common.Address, tokenId []*big.Int, buyer []common.Address, handler EventHandler) (ListenerHandle, error) {
	el.mu.Lock()
	defer el.mu.Unlock()

	subID := uuid.New().String()

	btContract, err := el.getOrCreateBookingToken(btAddress)
	if err != nil {
		return nil, err
	}

	ctx, cancel := context.WithCancel(context.Background())

	subInfo := &subscriptionInfo{
		subID:       subID,
		contract:    btAddress,
		handler:     handler,
		unsubscribe: cancel,
	}
	el.subscriptions[subID] = subInfo

	go el.resubscribeTokenBought(ctx, subID, btContract, tokenId, buyer)

	return &listenerHandle{
		unsubscribe: func() {
			el.unsubscribe(subID)
		},
	}, nil
}

// resubscribeTokenBought handles resubscription for TokenBought events
func (el *EventListener) resubscribeTokenBought(ctx context.Context, subID string, btContract *bookingtoken.Bookingtoken, tokenId []*big.Int, buyer []common.Address) {
	backoffMax := 2 * time.Minute

	resubscribeFn := func(ctx context.Context, lastError error) (event.Subscription, error) {
		if lastError != nil {
			el.logger.Errorf("Resubscribe attempt after error: %v", lastError)
		}

		eventChan := make(chan *bookingtoken.BookingtokenTokenBought)

		sub, err := btContract.WatchTokenBought(&bind.WatchOpts{Context: ctx}, eventChan, tokenId, buyer)
		if err != nil {
			el.logger.Errorf("Failed to subscribe to TokenBought events: %v", err)
			return nil, err
		}

		el.mu.Lock()
		subInfo, exists := el.subscriptions[subID]
		if !exists {
			el.mu.Unlock()
			sub.Unsubscribe()
			return nil, fmt.Errorf("subscription %s no longer exists", subID)
		}
		subInfo.sub = sub
		subInfo.closeChan = func() {
			close(eventChan)
		}
		el.mu.Unlock()

		go el.listenForTokenBoughtEvents(subID, eventChan)

		el.logger.Infof("Subscribed for TokenBought events on BookingToken %s", subInfo.contract)
		el.logger.Debugf("Subscription ID: %s", subID)
		el.logger.Debugf("Filters: tokenId: %v, buyer: %v", tokenId, buyer)

		return sub, nil
	}

	sub := event.ResubscribeErr(backoffMax, resubscribeFn)

	select {
	case <-ctx.Done():
		sub.Unsubscribe()
	}
}

// listenForTokenBoughtEvents listens for TokenBought events
func (el *EventListener) listenForTokenBoughtEvents(subID string, eventChan chan *bookingtoken.BookingtokenTokenBought) {
	el.mu.Lock()
	subInfo, exists := el.subscriptions[subID]
	el.mu.Unlock()
	if !exists {
		return
	}
	handler := subInfo.handler

	for {
		select {
		case event, ok := <-eventChan:
			if !ok {
				return
			}
			handler(event)
		}
	}
}

// RegisterTokenReservedHandler registers a handler for TokenReserved events on a BookingToken contract
func (el *EventListener) RegisterTokenReservedHandler(btAddress common.Address, tokenId []*big.Int, reservedFor []common.Address, supplier []common.Address, handler EventHandler) (ListenerHandle, error) {
	el.mu.Lock()
	defer el.mu.Unlock()

	subID := uuid.New().String()

	btContract, err := el.getOrCreateBookingToken(btAddress)
	if err != nil {
		return nil, err
	}

	ctx, cancel := context.WithCancel(context.Background())

	subInfo := &subscriptionInfo{
		subID:       subID,
		contract:    btAddress,
		handler:     handler,
		unsubscribe: cancel,
	}
	el.subscriptions[subID] = subInfo

	go el.resubscribeTokenReserved(ctx, subID, btContract, tokenId, reservedFor, supplier)

	return &listenerHandle{
		unsubscribe: func() {
			el.unsubscribe(subID)
		},
	}, nil
}

// resubscribeTokenReserved handles resubscription for TokenReserved events
func (el *EventListener) resubscribeTokenReserved(ctx context.Context, subID string, btContract *bookingtoken.Bookingtoken, tokenId []*big.Int, reservedFor []common.Address, supplier []common.Address) {
	backoffMax := 2 * time.Minute

	resubscribeFn := func(ctx context.Context, lastError error) (event.Subscription, error) {
		if lastError != nil {
			el.logger.Errorf("Resubscribe attempt after error: %v", lastError)
		}

		eventChan := make(chan *bookingtoken.BookingtokenTokenReserved)

		sub, err := btContract.WatchTokenReserved(&bind.WatchOpts{Context: ctx}, eventChan, tokenId, reservedFor, supplier)
		if err != nil {
			el.logger.Errorf("Failed to subscribe to TokenReserved events: %v", err)
			return nil, err
		}

		el.mu.Lock()
		subInfo, exists := el.subscriptions[subID]
		if !exists {
			el.mu.Unlock()
			sub.Unsubscribe()
			return nil, fmt.Errorf("subscription %s no longer exists", subID)
		}
		subInfo.sub = sub
		subInfo.closeChan = func() {
			close(eventChan)
		}
		el.mu.Unlock()

		go el.listenForTokenReservedEvents(subID, eventChan)

		el.logger.Infof("Subscribed for TokenReserved events on BookingToken %s", subInfo.contract)
		el.logger.Debugf("Subscription ID: %s", subID)
		el.logger.Debugf("Filters: tokenId: %v, reservedFor: %v, supplier: %v", tokenId, reservedFor, supplier)

		return sub, nil
	}

	sub := event.ResubscribeErr(backoffMax, resubscribeFn)

	select {
	case <-ctx.Done():
		sub.Unsubscribe()
	}
}

// listenForTokenReservedEvents listens for TokenReserved events
func (el *EventListener) listenForTokenReservedEvents(subID string, eventChan chan *bookingtoken.BookingtokenTokenReserved) {
	el.mu.Lock()
	subInfo, exists := el.subscriptions[subID]
	el.mu.Unlock()
	if !exists {
		return
	}
	handler := subInfo.handler

	for {
		select {
		case event, ok := <-eventChan:
			if !ok {
				return
			}
			handler(event)
		}
	}
}
