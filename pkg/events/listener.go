package events

import (
	"context"
	"fmt"
	"math/big"
	"sync"
	"time"

	"github.com/chain4travel/camino-messenger-contracts/go/contracts/bookingtokenv2"
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
	mu            sync.RWMutex
	cmAccounts    map[common.Address]*cmaccount.Cmaccount
	btContracts   map[common.Address]*bookingtokenv2.Bookingtokenv2
	subscriptions map[string]*subscriptionInfo // Keyed by unique IDs
}

// NewEventListener creates a new EventListener instance
func NewEventListener(client *ethclient.Client, logger *zap.SugaredLogger) *EventListener {
	return &EventListener{
		client:        client,
		logger:        logger,
		cmAccounts:    make(map[common.Address]*cmaccount.Cmaccount),
		btContracts:   make(map[common.Address]*bookingtokenv2.Bookingtokenv2),
		subscriptions: make(map[string]*subscriptionInfo),
	}
}

// getOrCreateCMAccount gets or creates a CMAccount instance
func (el *EventListener) getOrCreateCMAccount(addr common.Address) (*cmaccount.Cmaccount, error) {
	el.mu.RLock()
	cm, exists := el.cmAccounts[addr]
	el.mu.RUnlock()
	if exists {
		return cm, nil
	}

	cm, err := cmaccount.NewCmaccount(addr, el.client)
	if err != nil {
		return nil, fmt.Errorf("failed to create CMAccount instance: %w", err)
	}

	el.mu.Lock()
	el.cmAccounts[addr] = cm
	el.mu.Unlock()
	return cm, nil
}

// getOrCreateBookingToken gets or creates a BookingToken instance
func (el *EventListener) getOrCreateBookingToken(addr common.Address) (*bookingtokenv2.Bookingtokenv2, error) {
	el.mu.RLock()
	bt, exists := el.btContracts[addr]
	el.mu.RUnlock()
	if exists {
		return bt, nil
	}

	bt, err := bookingtokenv2.NewBookingtokenv2(addr, el.client)
	if err != nil {
		return nil, fmt.Errorf("failed to create BookingToken instance: %w", err)
	}

	el.mu.Lock()
	el.btContracts[addr] = bt
	el.mu.Unlock()
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

// getSubscription gets a subscription from the subscriptions map
func (el *EventListener) getSubscriptionInfo(subID string) (*subscriptionInfo, bool) {
	el.mu.RLock()
	defer el.mu.RUnlock()
	subInfo, exists := el.subscriptions[subID]
	return subInfo, exists
}

// addSubscription adds a subscription to the subscriptions map
func (el *EventListener) addSubscriptionInfo(subID string, subInfo *subscriptionInfo) {
	el.mu.Lock()
	defer el.mu.Unlock()
	el.subscriptions[subID] = subInfo
}

// addSubAndCloseChan adds a subscription and a close channel to the subscriptions map
func (el *EventListener) addSubAndCloseChan(subInfo *subscriptionInfo, sub event.Subscription, closeChan func()) {
	el.mu.Lock()
	defer el.mu.Unlock()

	subInfo.sub = sub
	subInfo.closeChan = closeChan
}

// RegisterServiceAddedHandler registers a handler for the ServiceAdded event on a CMAccount
func (el *EventListener) RegisterServiceAddedHandler(cmAccountAddr common.Address, serviceName []string, handler EventHandler) (func(), error) {
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
	el.addSubscriptionInfo(subID, subInfo)

	// Start the resubscription
	go el.resubscribeServiceAdded(ctx, subID, cmAccount, serviceName)

	// Return an unsubscribe function to stop listening
	return func() {
		el.unsubscribe(subID)
	}, nil
}

// resubscribeServiceAdded handles resubscription for ServiceAdded events
func (el *EventListener) resubscribeServiceAdded(ctx context.Context, subID string, cmAccount *cmaccount.Cmaccount, serviceName []string) {
	// TODO: Maybe this should be configurable
	backoffMax := 2 * time.Minute // Maximum backoff time between retries

	resubscribeFn := func(ctx context.Context, lastError error) (event.Subscription, error) {
		if lastError != nil {
			el.logger.Errorf("Resubscribe attempt after error: %v", lastError)
		}

		eventChan := make(chan *cmaccount.CmaccountServiceAdded)

		subInfo, exists := el.getSubscriptionInfo(subID)
		if !exists {
			return nil, fmt.Errorf("subscription %s no longer exists", subID)
		}

		// Subscribe to the event
		sub, err := cmAccount.WatchServiceAdded(&bind.WatchOpts{Context: ctx}, eventChan, serviceName)
		if err != nil {
			el.logger.Errorf("Failed to subscribe to ServiceAdded events: %v", err)
			return nil, err
		}

		el.addSubAndCloseChan(subInfo, sub, func() {
			close(eventChan)
		})

		go el.listenForServiceAddedEvents(subID, eventChan)

		el.logger.Infof("[SUB][ServiceAdded] CMAccount: %s SubID: %s [Filters] ServiceName: %v", subInfo.contract, subID, serviceName)

		return sub, nil
	}

	// Start resubscribing
	sub := event.ResubscribeErr(backoffMax, resubscribeFn)

	// Wait until context is canceled
	<-ctx.Done()
	sub.Unsubscribe()
}

// listenForServiceAddedEvents listens for ServiceAdded events
func (el *EventListener) listenForServiceAddedEvents(subID string, eventChan chan *cmaccount.CmaccountServiceAdded) {
	subInfo, exists := el.getSubscriptionInfo(subID)
	if !exists {
		return
	}

	handler := subInfo.handler

	for event := range eventChan {
		handler(event)
	}
}

// RegisterServiceFeeUpdatedHandler registers a handler for the ServiceFeeUpdated event on a CMAccount
func (el *EventListener) RegisterServiceFeeUpdatedHandler(cmAccountAddr common.Address, serviceName []string, handler EventHandler) (func(), error) {
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
	el.addSubscriptionInfo(subID, subInfo)

	go el.resubscribeServiceFeeUpdated(ctx, subID, cmAccount, serviceName)

	// Return an unsubscribe function to stop listening
	return func() {
		el.unsubscribe(subID)
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

		subInfo, exists := el.getSubscriptionInfo(subID)
		if !exists {
			return nil, fmt.Errorf("subscription %s no longer exists", subID)
		}

		sub, err := cmAccount.WatchServiceFeeUpdated(&bind.WatchOpts{Context: ctx}, eventChan, serviceName)
		if err != nil {
			el.logger.Errorf("Failed to subscribe to ServiceFeeUpdated events: %v", err)
			return nil, err
		}

		el.addSubAndCloseChan(subInfo, sub, func() {
			close(eventChan)
		})

		go el.listenForServiceFeeUpdatedEvents(subID, eventChan)

		el.logger.Infof("[SUB][ServiceFeeUpdated] CMAccount: %s SubID: %s [Filters] ServiceName: %v", subInfo.contract, subID, serviceName)

		return sub, nil
	}

	sub := event.ResubscribeErr(backoffMax, resubscribeFn)

	// Wait until context is canceled
	<-ctx.Done()
	sub.Unsubscribe()
}

// listenForServiceFeeUpdatedEvents listens for ServiceFeeUpdated events
func (el *EventListener) listenForServiceFeeUpdatedEvents(subID string, eventChan chan *cmaccount.CmaccountServiceFeeUpdated) {
	subInfo, exists := el.getSubscriptionInfo(subID)
	if !exists {
		return
	}

	handler := subInfo.handler

	for event := range eventChan {
		handler(event)
	}
}

// RegisterServiceRemovedHandler registers a handler for the ServiceRemoved event on a CMAccount
func (el *EventListener) RegisterServiceRemovedHandler(cmAccountAddr common.Address, serviceName []string, handler EventHandler) (func(), error) {
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
	el.addSubscriptionInfo(subID, subInfo)

	go el.resubscribeServiceRemoved(ctx, subID, cmAccount, serviceName)

	// Return an unsubscribe function to stop listening
	return func() {
		el.unsubscribe(subID)
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

		subInfo, exists := el.getSubscriptionInfo(subID)
		if !exists {
			return nil, fmt.Errorf("subscription %s no longer exists", subID)
		}

		sub, err := cmAccount.WatchServiceRemoved(&bind.WatchOpts{Context: ctx}, eventChan, serviceName)
		if err != nil {
			el.logger.Errorf("Failed to subscribe to ServiceRemoved events: %v", err)
			return nil, err
		}

		el.addSubAndCloseChan(subInfo, sub, func() {
			close(eventChan)
		})

		go el.listenForServiceRemovedEvents(subID, eventChan)

		el.logger.Infof("[SUB][ServiceRemoved] CMAccount: %s SubID: %s [Filters] ServiceName: %v", subInfo.contract, subID, serviceName)

		return sub, nil
	}

	sub := event.ResubscribeErr(backoffMax, resubscribeFn)

	// Wait until context is canceled
	<-ctx.Done()
	sub.Unsubscribe()
}

// listenForServiceRemovedEvents listens for ServiceRemoved events
func (el *EventListener) listenForServiceRemovedEvents(subID string, eventChan chan *cmaccount.CmaccountServiceRemoved) {
	subInfo, exists := el.getSubscriptionInfo(subID)
	if !exists {
		return
	}

	handler := subInfo.handler

	for event := range eventChan {
		handler(event)
	}
}

// RegisterCMAccountUpgradedHandler registers a handler for the CMAccountUpgraded event on a CMAccount
func (el *EventListener) RegisterCMAccountUpgradedHandler(cmAccountAddr common.Address, oldImplementation []common.Address, newImplementation []common.Address, handler EventHandler) (func(), error) {
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
	el.addSubscriptionInfo(subID, subInfo)

	go el.resubscribeCMAccountUpgraded(ctx, subID, cmAccount, oldImplementation, newImplementation)

	// Return an unsubscribe function to stop listening
	return func() {
		el.unsubscribe(subID)
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

		subInfo, exists := el.getSubscriptionInfo(subID)
		if !exists {
			return nil, fmt.Errorf("subscription %s no longer exists", subID)
		}

		sub, err := cmAccount.WatchCMAccountUpgraded(&bind.WatchOpts{Context: ctx}, eventChan, oldImplementation, newImplementation)
		if err != nil {
			el.logger.Errorf("Failed to subscribe to CMAccountUpgraded events: %v", err)
			return nil, err
		}

		el.addSubAndCloseChan(subInfo, sub, func() {
			close(eventChan)
		})

		go el.listenForCMAccountUpgradedEvents(subID, eventChan)

		el.logger.Infof("[SUB][CMAccountUpgraded] CMAccount: %s SubID: %s [Filters] oldImplementation: %v, newImplementation: %v", subInfo.contract, subID, oldImplementation, newImplementation)

		return sub, nil
	}

	sub := event.ResubscribeErr(backoffMax, resubscribeFn)

	// Wait until context is canceled
	<-ctx.Done()
	sub.Unsubscribe()
}

// listenForCMAccountUpgradedEvents listens for CMAccountUpgraded events
func (el *EventListener) listenForCMAccountUpgradedEvents(subID string, eventChan chan *cmaccount.CmaccountCMAccountUpgraded) {
	subInfo, exists := el.getSubscriptionInfo(subID)
	if !exists {
		return
	}

	handler := subInfo.handler

	for event := range eventChan {
		handler(event)
	}
}

// RegisterTokenBoughtHandler registers a handler for TokenBought events on a BookingToken contract
func (el *EventListener) RegisterTokenBoughtHandler(bookingTokenAddress common.Address, tokenID []*big.Int, buyer []common.Address, handler EventHandler) (func(), error) {
	subID := uuid.New().String()

	btContract, err := el.getOrCreateBookingToken(bookingTokenAddress)
	if err != nil {
		return nil, err
	}

	ctx, cancel := context.WithCancel(context.Background())

	subInfo := &subscriptionInfo{
		subID:       subID,
		contract:    bookingTokenAddress,
		handler:     handler,
		unsubscribe: cancel,
	}
	el.addSubscriptionInfo(subID, subInfo)

	go el.resubscribeTokenBought(ctx, subID, btContract, tokenID, buyer)

	// Return an unsubscribe function to stop listening
	return func() {
		el.unsubscribe(subID)
	}, nil
}

// resubscribeTokenBought handles resubscription for TokenBought events
func (el *EventListener) resubscribeTokenBought(ctx context.Context, subID string, btContract *bookingtokenv2.Bookingtokenv2, tokenID []*big.Int, buyer []common.Address) {
	backoffMax := 2 * time.Minute

	resubscribeFn := func(ctx context.Context, lastError error) (event.Subscription, error) {
		if lastError != nil {
			el.logger.Errorf("Resubscribe attempt after error: %v", lastError)
		}

		eventChan := make(chan *bookingtokenv2.Bookingtokenv2TokenBought)

		subInfo, exists := el.getSubscriptionInfo(subID)
		if !exists {
			return nil, fmt.Errorf("subscription %s no longer exists", subID)
		}

		sub, err := btContract.WatchTokenBought(&bind.WatchOpts{Context: ctx}, eventChan, tokenID, buyer)
		if err != nil {
			el.logger.Errorf("Failed to subscribe to TokenBought events: %v", err)
			return nil, err
		}

		el.addSubAndCloseChan(subInfo, sub, func() {
			close(eventChan)
		})

		go el.listenForTokenBoughtEvents(subID, eventChan)

		el.logger.Infof("[SUB][TokenBought] BookingToken: %s SubID: %s [Filters] tokenID: %v, buyer: %v", subInfo.contract, subID, tokenID, buyer)

		return sub, nil
	}

	sub := event.ResubscribeErr(backoffMax, resubscribeFn)

	// Wait until context is canceled
	<-ctx.Done()
	sub.Unsubscribe()
}

// listenForTokenBoughtEvents listens for TokenBought events
func (el *EventListener) listenForTokenBoughtEvents(subID string, eventChan chan *bookingtokenv2.Bookingtokenv2TokenBought) {
	subInfo, exists := el.getSubscriptionInfo(subID)
	if !exists {
		return
	}

	handler := subInfo.handler

	for event := range eventChan {
		handler(event)
	}
}

// RegisterTokenReservedHandler registers a handler for TokenReserved events on a BookingToken contract
func (el *EventListener) RegisterTokenReservedHandler(bookingTokenAddress common.Address, tokenID []*big.Int, reservedFor []common.Address, supplier []common.Address, handler EventHandler) (func(), error) {
	subID := uuid.New().String()

	btContract, err := el.getOrCreateBookingToken(bookingTokenAddress)
	if err != nil {
		return nil, err
	}

	ctx, cancel := context.WithCancel(context.Background())

	subInfo := &subscriptionInfo{
		subID:       subID,
		contract:    bookingTokenAddress,
		handler:     handler,
		unsubscribe: cancel,
	}
	el.addSubscriptionInfo(subID, subInfo)

	go el.resubscribeTokenReserved(ctx, subID, btContract, tokenID, reservedFor, supplier)

	// Return an unsubscribe function to stop listening
	return func() {
		el.unsubscribe(subID)
	}, nil
}

// resubscribeTokenReserved handles resubscription for TokenReserved events
func (el *EventListener) resubscribeTokenReserved(ctx context.Context, subID string, btContract *bookingtokenv2.Bookingtokenv2, tokenID []*big.Int, reservedFor []common.Address, supplier []common.Address) {
	backoffMax := 2 * time.Minute

	resubscribeFn := func(ctx context.Context, lastError error) (event.Subscription, error) {
		if lastError != nil {
			el.logger.Errorf("Resubscribe attempt after error: %v", lastError)
		}

		eventChan := make(chan *bookingtokenv2.Bookingtokenv2TokenReserved)

		subInfo, exists := el.getSubscriptionInfo(subID)
		if !exists {
			return nil, fmt.Errorf("subscription %s no longer exists", subID)
		}

		sub, err := btContract.WatchTokenReserved(&bind.WatchOpts{Context: ctx}, eventChan, tokenID, reservedFor, supplier)
		if err != nil {
			el.logger.Errorf("Failed to subscribe to TokenReserved events: %v", err)
			return nil, err
		}

		el.addSubAndCloseChan(subInfo, sub, func() {
			close(eventChan)
		})

		go el.listenForTokenReservedEvents(subID, eventChan)

		el.logger.Infof("[SUB][TokenReserved] BookingToken: %s SubID: %s [Filters] tokenID: %v, reservedFor: %v, supplier: %v", subInfo.contract, subID, tokenID, reservedFor, supplier)

		return sub, nil
	}

	sub := event.ResubscribeErr(backoffMax, resubscribeFn)

	// Wait until context is canceled
	<-ctx.Done()
	sub.Unsubscribe()
}

// listenForTokenReservedEvents listens for TokenReserved events
func (el *EventListener) listenForTokenReservedEvents(subID string, eventChan chan *bookingtokenv2.Bookingtokenv2TokenReserved) {
	subInfo, exists := el.getSubscriptionInfo(subID)
	if !exists {
		return
	}

	handler := subInfo.handler

	for event := range eventChan {
		handler(event)
	}
}

// RegisterChequeCashedInHandler registers a handler for the ChequeCashedIn event on a CMAccount
func (el *EventListener) RegisterChequeCashedInHandler(cmAccountAddr common.Address, fromAccounts, toAccounts []common.Address, handler EventHandler) (func(), error) {
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
	el.addSubscriptionInfo(subID, subInfo)

	// Start the resubscription
	go el.resubscribeChequeCashedIn(ctx, subID, cmAccount, fromAccounts, toAccounts)

	// Return an unsubscribe function to stop listening
	return func() {
		el.unsubscribe(subID)
	}, nil
}

// resubscribeChequeCashedIn handles resubscription for ChequeCashedIn events
func (el *EventListener) resubscribeChequeCashedIn(ctx context.Context, subID string, cmAccount *cmaccount.Cmaccount, fromAccounts, toAccounts []common.Address) {
	backoffMax := 2 * time.Minute // Maximum backoff time between retries

	resubscribeFn := func(ctx context.Context, lastError error) (event.Subscription, error) {
		if lastError != nil {
			el.logger.Errorf("Resubscribe attempt after error: %v", lastError)
		}

		eventChan := make(chan *cmaccount.CmaccountChequeCashedIn)

		subInfo, exists := el.getSubscriptionInfo(subID)
		if !exists {
			return nil, fmt.Errorf("subscription %s no longer exists", subID)
		}

		// Subscribe to the event
		sub, err := cmAccount.WatchChequeCashedIn(&bind.WatchOpts{Context: ctx}, eventChan, fromAccounts, toAccounts)
		if err != nil {
			el.logger.Errorf("Failed to subscribe to ChequeCashedIn events: %v", err)
			return nil, err
		}

		el.addSubAndCloseChan(subInfo, sub, func() {
			close(eventChan)
		})

		go el.listenForChequeCashedInEvents(subID, eventChan)

		el.logger.Infof("[SUB][ChequeCashedIn] CMAccount: %s SubID: %s [Filters] (fromAccounts, toAccounts): (%v, %v)", subInfo.contract, subID, fromAccounts, toAccounts)

		return sub, nil
	}

	// Start resubscribing
	sub := event.ResubscribeErr(backoffMax, resubscribeFn)

	// Wait until context is canceled
	<-ctx.Done()
	sub.Unsubscribe()
}

// listenForChequeCashedInEvents listens for ChequeCashedIn events
func (el *EventListener) listenForChequeCashedInEvents(subID string, eventChan chan *cmaccount.CmaccountChequeCashedIn) {
	subInfo, exists := el.getSubscriptionInfo(subID)
	if !exists {
		return
	}

	handler := subInfo.handler

	for event := range eventChan {
		handler(event)
	}
}

// TODO: @VjeraTurk
// RegisterCancellationPendingHandler
// resubscribeCancellationPending
// listenForCancellationPendingEvents

// RegisterCancellationAcceptedHandler
// resubscribeCancellationAccepted
// listenForCancellationAcceptedEvents

// RegisterCancellationRejectedHandler
// resubscribeCancellationRejected
// listenForCancellationRejectedEvents

// RegisterCancellationCounteredHandler
// resubscribeCancellationCountered
// listenForCancellationCounteredEvents

// RegisterTokenCancellableUpdated
// resubscribeTokenCancellableUpdated
// listenForTokenCancellableUpdated
