// Copyright (C) 2022-2025, Chain4Travel AG. All rights reserved.
// See the file LICENSE for licensing terms.

package eventlistener

import (
	"context"
	"errors"
	"math/big"
	"sync"
	"time"

	"github.com/chain4travel/camino-messenger-bot/v11/internal/event_listener/subscriber"
	"github.com/chain4travel/camino-messenger-bot/v11/internal/partnerplugin"
	"github.com/chain4travel/camino-messenger-bot/v11/pkg/booking"
	cmaccounts "github.com/chain4travel/camino-messenger-bot/v11/pkg/cm_accounts"
	cmbcommon "github.com/chain4travel/camino-messenger-bot/v11/pkg/cmbcommon"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/ethclient"
	"go.uber.org/zap"
)

const timeCheckBlockchainDelay = 2 * time.Second

var (
	_ EventListener = (*eventListener)(nil)

	maxTime = time.Unix(int64(1<<63-cmbcommon.TimePkgOffset), 0)

	ErrNotFound = errors.New("not found")
)

type Storage interface {
	SessionHandler
	TokenBoughtSubscriptionStorage
}

type SessionHandler interface {
	NewSession(context.Context) (Session, error)
	Commit(Session) error
	Abort(Session)
}

type Session interface {
	Commit() error
	Abort() error
}

type EventListener interface {
	Start(context.Context) error
	Stop()
	TokenBoughtSubscriber
}

type eventListener struct {
	startingBlockNumber *big.Int

	storage        Storage
	logger         *zap.SugaredLogger
	subscriber     subscriber.Subscriber
	partnerPlugin  partnerplugin.PartnerPlugin
	bookingService booking.Service

	recordTokenExpiration        bool
	unsubscribeTokenBought       func()
	tokenBoughtTimerMutex        sync.Mutex
	tokenBoughtTimer             *time.Timer
	tokenBoughtTimerSubscription *TokenBoughtSubscription
}

func New(
	ctx context.Context,
	logger *zap.SugaredLogger,
	storage Storage,
	ethClient *ethclient.Client,
	bookingTokenAddress common.Address,
	partnerPlugin partnerplugin.PartnerPlugin,
	bookingService booking.Service,
	cmAccounts cmaccounts.Service,
	recordExpiration bool,
) (EventListener, error) {
	blockNumber, err := ethClient.BlockNumber(ctx)
	if err != nil {
		logger.Errorf("failed to get latest block number: %v", err)
		return nil, err
	}

	subscriber, err := subscriber.New(ethClient, logger, bookingTokenAddress, cmAccounts, blockNumber)
	if err != nil {
		logger.Errorf("failed to create subscriber: %v", err)
		return nil, err
	}

	return &eventListener{
		logger:                logger,
		storage:               storage,
		subscriber:            subscriber,
		partnerPlugin:         partnerPlugin,
		bookingService:        bookingService,
		recordTokenExpiration: recordExpiration,
		startingBlockNumber:   big.NewInt(0).SetUint64(blockNumber),
	}, nil
}

func (l *eventListener) Start(ctx context.Context) error {
	if err := l.startTokenBoughtSubscriptions(ctx); err != nil {
		l.logger.Errorf("failed to start token bought subscriptions: %v", err)
		return err
	}
	return nil
}

func (l *eventListener) Stop() {
	l.unsubscribeTokenBought()
	l.stopTokenBoughtTimer()
}
