// Copyright (C) 2022-2025, Chain4Travel AG. All rights reserved.
// See the file LICENSE for licensing terms.

package eventlistener

import (
	"errors"
	"math/big"
	"sync"
	"time"

	"github.com/chain4travel/camino-messenger-bot/internal/event_listener/subscriber"
	"github.com/chain4travel/camino-messenger-bot/internal/partnerplugin"
	cmaccounts "github.com/chain4travel/camino-messenger-bot/pkg/cm_accounts"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/ethclient"
	"go.uber.org/zap"
)

var (
	_ EventListener = (*eventListener)(nil)

	ErrNotFound = errors.New("not found")
)

type EventListener interface {
	Stop()
	SubscribeForTokenBoughtEvent(tokenID *big.Int, mintID string, timeout time.Time) error
}

type eventListener struct {
	bookingTokenAddress common.Address
	logger              *zap.SugaredLogger
	subscriber          subscriber.Subscriber
	partnerPlugin       partnerplugin.PartnerPlugin

	unsubscribers      []unsubscriber
	unsubscribersMutex sync.Mutex
}

type unsubscriber struct {
	unsubscribe  func()
	timeoutTimer *time.Timer
}

func New(
	logger *zap.SugaredLogger,
	ethClient *ethclient.Client,
	bookingTokenAddress common.Address,
	cmAccounts cmaccounts.Service,
	partnerPlugin partnerplugin.PartnerPlugin,
) (EventListener, error) {
	subscriber, err := subscriber.New(ethClient, logger, bookingTokenAddress, cmAccounts)
	if err != nil {
		logger.Errorf("failed to create subscriber: %v", err)
		return nil, err
	}

	return &eventListener{
		bookingTokenAddress: bookingTokenAddress,
		logger:              logger,
		subscriber:          subscriber,
		partnerPlugin:       partnerPlugin,
	}, nil
}

func (el *eventListener) Stop() {
	el.unsubscribersMutex.Lock()
	defer el.unsubscribersMutex.Unlock()
	for _, subscription := range el.unsubscribers {
		subscription.unsubscribe()
		subscription.timeoutTimer.Stop()
	}
}

func (el *eventListener) addUnsubscriber(unsubscriber unsubscriber) {
	el.unsubscribersMutex.Lock()
	defer el.unsubscribersMutex.Unlock()
	el.unsubscribers = append(el.unsubscribers, unsubscriber)
}
