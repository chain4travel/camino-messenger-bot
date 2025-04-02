package eventlistener

import (
	"context"
	"errors"
	"math/big"
	"time"

	partnerplugin "github.com/chain4travel/camino-messenger-bot/internal/partner_plugin"
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
	SubscribeForTokenBoughtEvent(ctx context.Context, tokenID *big.Int, mintID string, timeout time.Time) error
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

func (el *eventListener) Stop(ctx context.Context) {
	for _, subscription := range el.unsubscribers {
		subscription.unsubscribe()
		subscription.timeoutTimer.Stop()
	}
}
