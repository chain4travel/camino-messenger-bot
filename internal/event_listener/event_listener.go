// Copyright (C) 2022-2025, Chain4Travel AG. All rights reserved.
// See the file LICENSE for licensing terms.

package eventlistener

import (
	"errors"
	"math/big"
	"time"

	"github.com/chain4travel/camino-messenger-bot/internal/partnerplugin"
	"github.com/chain4travel/camino-messenger-bot/pkg/events"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/ethclient"
	"go.uber.org/zap"
)

var (
	_ EventListener = (*eventListener)(nil)

	ErrNotFound = errors.New("not found")
)

type EventListener interface {
	SubscribeForTokenBoughtEvent(tokenID *big.Int, mintID string, timeout time.Time) error
}

type eventListener struct {
	bookingTokenAddress common.Address
	logger              *zap.SugaredLogger
	eventListener       *events.EventListener
	partnerPlugin       partnerplugin.PartnerPlugin

	unsubscribers []unsubscriber
}

type unsubscriber struct {
	unsubscribe  func()
	timeoutTimer *time.Timer
}

func New(
	logger *zap.SugaredLogger,
	ethClient *ethclient.Client,
	bookingTokenAddress common.Address,
	partnerPlugin partnerplugin.PartnerPlugin,
) EventListener {
	return &eventListener{
		bookingTokenAddress: bookingTokenAddress,
		logger:              logger,
		eventListener:       events.NewEventListener(ethClient, logger),
		partnerPlugin:       partnerPlugin,
	}
}

func (el *eventListener) Stop() {
	for _, subscription := range el.unsubscribers {
		subscription.unsubscribe()
		subscription.timeoutTimer.Stop()
	}
}
