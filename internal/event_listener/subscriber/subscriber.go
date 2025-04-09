// Copyright (C) 2022-2025, Chain4Travel AG. All rights reserved.
// See the file LICENSE for licensing terms.

package subscriber

import (
	"context"
	"math/big"
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
		handler func(*cmaccount.CmaccountServiceAdded),
	) (unsubscribe func(), err error)

	SubscribeTokenBought(
		tokenID *big.Int,
		handler func(*bookingtoken.BookingtokenTokenBought),
	) (unsubscribe func())
}

type subscriber struct {
	client       *ethclient.Client
	logger       *zap.SugaredLogger
	bookingToken *bookingtoken.Bookingtoken
	cmAccounts   cmaccounts.Service
}

func New(
	client *ethclient.Client,
	logger *zap.SugaredLogger,
	bookingTokenAddress common.Address,
	cmAccounts cmaccounts.Service,
) (Subscriber, error) {
	bookingToken, err := bookingtoken.NewBookingtoken(bookingTokenAddress, client)
	if err != nil {
		logger.Errorf("failed to create booking token contract binding: %v", err)
		return nil, err
	}

	return &subscriber{
		client:       client,
		logger:       logger,
		bookingToken: bookingToken,
		cmAccounts:   cmAccounts,
	}, nil
}

// Subscribes to the ServiceAdded event.
//
// [cmAccountAddr] is the address of the CMAccount contract.
//
// [handler] is the function to call when the event is triggered.
//
// Returns a function to unsubscribe from the event.
func (s *subscriber) SubscribeServiceAdded(
	cmAccountAddr common.Address,
	handler func(*cmaccount.CmaccountServiceAdded),
) (unsubscribe func(), err error) {
	cmAccount, err := s.cmAccounts.CMAccount(cmAccountAddr)
	if err != nil {
		return nil, err
	}

	return startResubscriber(
		s,
		handler,
		func(ctx context.Context, eventChan chan *cmaccount.CmaccountServiceAdded) (event.Subscription, error) {
			return cmAccount.WatchServiceAdded(&bind.WatchOpts{Context: ctx}, eventChan, nil)
		},
	), nil
}

// Subscribes to the TokenBought event.
//
// [handler] is the function to call when the event is triggered.
//
// Returns a function to unsubscribe from the event.
func (s *subscriber) SubscribeTokenBought(
	tokenID *big.Int,
	handler func(*bookingtoken.BookingtokenTokenBought),
) (unsubscribe func()) {
	return startResubscriber(
		s,
		handler,
		func(ctx context.Context, eventChan chan *bookingtoken.BookingtokenTokenBought) (event.Subscription, error) {
			return s.bookingToken.WatchTokenBought(&bind.WatchOpts{Context: ctx}, eventChan, []*big.Int{tokenID}, nil)
		},
	)
}

func startResubscriber[T any](
	s *subscriber,
	handler func(T),
	subscribe func(context.Context, chan T) (event.Subscription, error),
) func() {
	eventType := new(T) // for logging purposes

	eventChan := make(chan T)
	go func() {
		for event := range eventChan {
			handler(event)
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
