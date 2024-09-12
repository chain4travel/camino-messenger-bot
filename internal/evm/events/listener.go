package events

import (
	"context"
	"fmt"
	"log"
	"sync"

	"github.com/chain4travel/camino-messenger-contracts/go/contracts/bookingtoken"
	"github.com/chain4travel/camino-messenger-contracts/go/contracts/cmaccount"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/ethereum/go-ethereum/event"
)

// EventRegistry holds event handlers for both contracts
type EventRegistry struct {
	mu       sync.Mutex
	handlers map[string]map[string]EventHandler // Map of contract address -> event name -> handler
}

// EventHandler defines the type for custom event handlers
type EventHandler func(event interface{})

// NewEventRegistry creates a new registry for events
func NewEventRegistry() *EventRegistry {
	return &EventRegistry{
		handlers: make(map[string]map[string]EventHandler),
	}
}

// RegisterHandler registers a custom handler for a specific contract address and event name
func (e *EventRegistry) RegisterHandler(contractAddr common.Address, eventName string, handler EventHandler) {
	e.mu.Lock()
	defer e.mu.Unlock()

	addressStr := contractAddr.Hex()

	if _, exists := e.handlers[addressStr]; !exists {
		e.handlers[addressStr] = make(map[string]EventHandler)
	}

	e.handlers[addressStr][eventName] = handler
}

// UnregisterHandler removes the handler for a specific contract address and event name
func (e *EventRegistry) UnregisterHandler(contractAddr common.Address, eventName string) {
	e.mu.Lock()
	defer e.mu.Unlock()

	addressStr := contractAddr.Hex()

	if handlers, exists := e.handlers[addressStr]; exists {
		delete(handlers, eventName)

		// Clean up if no handlers left for the contract
		if len(handlers) == 0 {
			delete(e.handlers, addressStr)
		}
	}
}

// TriggerHandler triggers the handler for a specific contract address and event name if registered
func (e *EventRegistry) TriggerHandler(contractAddr common.Address, eventName string, event interface{}) {
	e.mu.Lock()
	defer e.mu.Unlock()

	addressStr := contractAddr.Hex()

	if handlers, exists := e.handlers[addressStr]; exists {
		if handler, exists := handlers[eventName]; exists {
			handler(event)
		}
	}
}

// EventListener listens for events from multiple deployments of CMAccount contracts and BookingToken
type EventListener struct {
	client                  *ethclient.Client
	bookingToken            *bookingtoken.Bookingtoken
	cmAccounts              map[common.Address]*cmaccount.Cmaccount
	eventRegistry           *EventRegistry
	tokenBoughtChan         chan *bookingtoken.BookingtokenTokenBought
	cmAccountEventChannels  map[common.Address]chan *cmaccount.CmaccountServiceAdded // Map of CMAccount address -> event channels
	cmAccountSubscriptions  map[common.Address]event.Subscription                    // Map of CMAccount address -> subscription
	tokenBoughtSubscription event.Subscription                                       // Subscription for BookingToken
}

// NewEventListener creates a new EventListener instance
func NewEventListener(client *ethclient.Client, bookingTokenAddr common.Address) (*EventListener, error) {
	bookingToken, err := bookingtoken.NewBookingtoken(bookingTokenAddr, client)
	if err != nil {
		return nil, fmt.Errorf("failed to instantiate Bookingtoken contract: %w", err)
	}

	return &EventListener{
		client:                 client,
		bookingToken:           bookingToken,
		cmAccounts:             make(map[common.Address]*cmaccount.Cmaccount),
		eventRegistry:          NewEventRegistry(),
		tokenBoughtChan:        make(chan *bookingtoken.BookingtokenTokenBought),
		cmAccountEventChannels: make(map[common.Address]chan *cmaccount.CmaccountServiceAdded),
		cmAccountSubscriptions: make(map[common.Address]event.Subscription),
	}, nil
}

// AddCMAccount adds a new CMAccount contract to listen for its events
func (el *EventListener) AddCMAccount(cmAccountAddr common.Address) error {
	cmAccount, err := cmaccount.NewCmaccount(cmAccountAddr, el.client)
	if err != nil {
		return fmt.Errorf("failed to instantiate CMAccount contract at %s: %w", cmAccountAddr.Hex(), err)
	}

	// Store CMAccount and create a channel for its events
	el.cmAccounts[cmAccountAddr] = cmAccount
	el.cmAccountEventChannels[cmAccountAddr] = make(chan *cmaccount.CmaccountServiceAdded)

	// Set up event watchers for this CMAccount
	opts := &bind.WatchOpts{Context: context.Background(), Start: nil}

	subscription, err := cmAccount.WatchServiceAdded(opts, el.cmAccountEventChannels[cmAccountAddr])
	if err != nil {
		return fmt.Errorf("failed to subscribe to ServiceAdded events for CMAccount at %s: %w", cmAccountAddr.Hex(), err)
	}

	el.cmAccountSubscriptions[cmAccountAddr] = subscription

	// Start listening for events for this specific CMAccount
	go el.listenForCMAccountEvents(cmAccountAddr)

	return nil
}

// RemoveCMAccount removes the event listener for a specific CMAccount and stops its subscription
func (el *EventListener) RemoveCMAccount(cmAccountAddr common.Address) {
	// Unsubscribe from the events
	if sub, exists := el.cmAccountSubscriptions[cmAccountAddr]; exists {
		sub.Unsubscribe()
		delete(el.cmAccountSubscriptions, cmAccountAddr)
	}

	// Remove event channel and CMAccount from the map
	delete(el.cmAccountEventChannels, cmAccountAddr)
	delete(el.cmAccounts, cmAccountAddr)

	// Unregister all event handlers related to this CMAccount
	el.eventRegistry.UnregisterHandler(cmAccountAddr, "ServiceAdded")
	el.eventRegistry.UnregisterHandler(cmAccountAddr, "ServiceFeeUpdated")
}

// StartListening starts the event listening for the BookingToken contract
func (el *EventListener) StartListening() error {
	opts := &bind.WatchOpts{Context: context.Background(), Start: nil}

	// Watch BookingToken TokenBought events
	tokenBoughtSub, err := el.bookingToken.WatchTokenBought(opts, el.tokenBoughtChan, nil, nil)
	if err != nil {
		return fmt.Errorf("failed to subscribe to TokenBought events: %w", err)
	}

	el.tokenBoughtSubscription = tokenBoughtSub
	go el.listenForBookingTokenEvents()

	return nil
}

// StopListening stops all event subscriptions, including BookingToken and CMAccounts
func (el *EventListener) StopListening() {
	// Stop BookingToken subscription
	if el.tokenBoughtSubscription != nil {
		el.tokenBoughtSubscription.Unsubscribe()
	}

	// Stop all CMAccount subscriptions
	for addr, subscription := range el.cmAccountSubscriptions {
		subscription.Unsubscribe()
		delete(el.cmAccountSubscriptions, addr)
	}
}

// RegisterHandler allows other parts of the app to register event handlers
func (el *EventListener) RegisterHandler(contractAddr common.Address, eventName string, handler EventHandler) {
	el.eventRegistry.RegisterHandler(contractAddr, eventName, handler)
}

// UnregisterHandler allows other parts of the app to unregister event handlers
func (el *EventListener) UnregisterHandler(contractAddr common.Address, eventName string) {
	el.eventRegistry.UnregisterHandler(contractAddr, eventName)
}

// listenForBookingTokenEvents listens for BookingToken events
func (el *EventListener) listenForBookingTokenEvents() {
	for {
		select {
		case event := <-el.tokenBoughtChan:
			el.eventRegistry.TriggerHandler(el.bookingToken.Address(), "TokenBought", event)
		}
	}
}

// listenForCMAccountEvents listens for events from a specific CMAccount
func (el *EventListener) listenForCMAccountEvents(cmAccountAddr common.Address) {
	for {
		select {
		case event := <-el.cmAccountEventChannels[cmAccountAddr]:
			el.eventRegistry.TriggerHandler(cmAccountAddr, "ServiceAdded", event)

		case err := <-el.cmAccountSubscriptions[cmAccountAddr].Err():
			log.Fatalf("Error in CMAccount event subscription at %s: %v", cmAccountAddr.Hex(), err)
		}
	}
}
