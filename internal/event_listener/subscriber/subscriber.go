// Copyright (C) 2022-2025, Chain4Travel AG. All rights reserved.
// See the file LICENSE for licensing terms.

package subscriber

import (
	"context"
	"sync/atomic"
	"time"

	cmaccounts "github.com/chain4travel/camino-messenger-bot/pkg/cm_accounts"
	"github.com/chain4travel/camino-messenger-contracts/go/contracts/bookingtoken"
	"github.com/chain4travel/camino-messenger-contracts/go/contracts/cmaccount"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/ethereum/go-ethereum/event"
	"go.uber.org/zap"
)

const backoffMax = 2 * time.Minute // Maximum backoff time between subscribe retries // TODO @havan Maybe this should be configurable

var _ Subscriber = (*subscriber)(nil)

type Subscriber interface {
	SubscribeServiceAdded(
		cmAccountAddr common.Address,
		handler func(*cmaccount.CmaccountServiceAdded) uint64,
	) (unsubscribe func(), err error)

	SubscribeTokenBought(
		handler func(*bookingtoken.BookingtokenTokenBought) uint64,
	) (unsubscribe func())
}

type subscriber struct {
	client       *ethclient.Client
	logger       *zap.SugaredLogger
	bookingToken *bookingtoken.Bookingtoken
	cmAccounts   cmaccounts.Service
	blockNumber  *atomic.Uint64
}

func New(
	client *ethclient.Client,
	logger *zap.SugaredLogger,
	bookingTokenAddress common.Address,
	cmAccounts cmaccounts.Service,
	blockNumber uint64,
) (Subscriber, error) {
	bookingToken, err := bookingtoken.NewBookingtoken(bookingTokenAddress, client)
	if err != nil {
		logger.Errorf("failed to create booking token contract binding: %v", err)
		return nil, err
	}

	blockNumberAtomic := &atomic.Uint64{}
	blockNumberAtomic.Store(blockNumber)

	return &subscriber{
		client:       client,
		logger:       logger,
		bookingToken: bookingToken,
		cmAccounts:   cmAccounts,
		blockNumber:  blockNumberAtomic,
	}, nil
}

// Subscribes to the ServiceAdded event.
//
// [fromBlockNumber] is the block number from which to start watching for events. If 0, it will start from the latest block.
//
// [cmAccountAddr] is the address of the CMAccount contract.
//
// [handler] is the function to call when the event is triggered.
// It receives the event as arguments and should return successfully processed block number or 0.
//
// Returns a function to unsubscribe from the event.
func (s *subscriber) SubscribeServiceAdded(
	cmAccountAddr common.Address,
	handler func(*cmaccount.CmaccountServiceAdded) uint64,
) (unsubscribe func(), err error) {
	cmAccount, err := s.cmAccounts.CMAccount(cmAccountAddr)
	if err != nil {
		return nil, err
	}

	return startResubscriber(
		s,
		handler,
		func(ctx context.Context, eventChan chan *cmaccount.CmaccountServiceAdded) (event.Subscription, error) {
			blockNumber := s.blockNumber.Load()
			return cmAccount.WatchServiceAdded(&bind.WatchOpts{Context: ctx, Start: &blockNumber}, eventChan, nil)
		},
	), nil
}

// Subscribes to the TokenBought event.
//
// [fromBlockNumber] is the block number from which to start watching for events. If 0, it will start from the latest block.
//
// [handler] is the function to call when the event is triggered.
// It receives the event as arguments and should return successfully processed block number or 0.
//
// Returns a function to unsubscribe from the event.
func (s *subscriber) SubscribeTokenBought(
	handler func(*bookingtoken.BookingtokenTokenBought) uint64,
) (unsubscribe func()) {
	return startResubscriber(
		s,
		handler,
		func(ctx context.Context, eventChan chan *bookingtoken.BookingtokenTokenBought) (event.Subscription, error) {
			blockNumber := s.blockNumber.Load()
			return s.bookingToken.WatchTokenBought(&bind.WatchOpts{Context: ctx, Start: &blockNumber}, eventChan, nil, nil)
		},
	)
}

func startResubscriber[T any](
	s *subscriber,
	handler func(T) uint64,
	subscribe func(context.Context, chan T) (event.Subscription, error),
) func() {
	eventType := new(T) // for logging purposes

	eventChan := make(chan T)
	go func() {
		for event := range eventChan {
			if successfullyProcessedBlockNumber := handler(event); successfullyProcessedBlockNumber != 0 {
				s.blockNumber.Store(successfullyProcessedBlockNumber)
			}
		}
	}()

	// ResubscribeErr starts the resubscription process in its own goroutine without blocking caller
	resubscriber := event.ResubscribeErr(backoffMax, func(ctx context.Context, lastError error) (event.Subscription, error) {
		if lastError != nil {
			s.logger.Errorf("Resubscribe attempt after error: %v", lastError)
		}

		sub, err := subscribe(ctx, eventChan)
		if err != nil {
			s.logger.Errorf("Failed to subscribe to %T events: %v", eventType, err)
			return nil, err
		}
		return sub, nil
	})

	return func() {
		resubscriber.Unsubscribe()
		if eventChan != nil {
			close(eventChan)
		}
	}
}
