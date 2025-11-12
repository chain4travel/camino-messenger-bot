// Copyright (C) 2022-2025, Chain4Travel AG. All rights reserved.
// See the file LICENSE for licensing terms.

package partnerplugin

import (
	"context"
	"fmt"
	"math/big"
	"time"

	"buf.build/gen/go/chain4travel/camino-messenger-protocol/grpc/go/cmp/services/notification/v3/notificationv3grpc"
	notificationv3 "buf.build/gen/go/chain4travel/camino-messenger-protocol/protocolbuffers/go/cmp/services/notification/v3"
	typesv4 "buf.build/gen/go/chain4travel/camino-messenger-protocol/protocolbuffers/go/cmp/types/v4"
	"buf.build/go/protovalidate"
	"github.com/ethereum/go-ethereum/common"
	"go.uber.org/zap"
	grpc_metadata "google.golang.org/grpc/metadata"

	types "github.com/chain4travel/camino-messenger-bot/v11/internal/messaging/types"
	rpc "github.com/chain4travel/camino-messenger-bot/v11/internal/rpc"
	"github.com/chain4travel/camino-messenger-bot/v11/internal/rpc/client"
	"github.com/chain4travel/camino-messenger-bot/v11/pkg/metadata"
)

var _ PartnerPlugin = (*partnerPlugin)(nil)

// Handles all communication with the partner plugin
type PartnerPlugin interface {
	DoServiceRequest(
		ctx context.Context,
		requestMsg *types.Message,
		serviceClient rpc.Client,
		fromCMAccount common.Address,
		toCMAccount common.Address,
	) (*types.Message, error)

	TokenBoughtNotificationWithoutBuyTx(ctx context.Context, tokenID *big.Int, mintID string) error
	TokenBoughtNotificationWithBuyTx(ctx context.Context, tokenID *big.Int, mintID string, buyTxID common.Hash) error
	TokenExpiredNotification(ctx context.Context, tokenID *big.Int, mintID string) error

	CancellationPendingNotification(ctx context.Context, notification *notificationv3.CancellationPending) error
	CancellationRejectedNotification(ctx context.Context, notification *notificationv3.CancellationRejected) error
	CancellationWithdrawnNotification(ctx context.Context, notification *notificationv3.CancellationWithdrawn) error
	CancellationFinalizedNotification(ctx context.Context, notification *notificationv3.CancellationFinalized) error
}

func New(
	logger *zap.SugaredLogger,
	rpcClient *client.RPCClient,
	responseTimeout time.Duration,
) PartnerPlugin {
	return &partnerPlugin{
		logger:             logger,
		notificationClient: notificationv3grpc.NewNotificationServiceClient(rpcClient.ClientConn),
		responseTimeout:    responseTimeout,
	}
}

type partnerPlugin struct {
	logger             *zap.SugaredLogger
	notificationClient notificationv3grpc.NotificationServiceClient
	responseTimeout    time.Duration
}

func (p *partnerPlugin) DoServiceRequest(
	ctx context.Context,
	requestMsg *types.Message,
	serviceClient rpc.Client,
	fromCMAccount common.Address,
	toCMAccount common.Address,
) (*types.Message, error) {
	responseMsg := &types.Message{
		RequestID:  requestMsg.RequestID,
		Timestamps: requestMsg.Timestamps,
	}

	var err error
	responseMsg.Content, responseMsg.Type, err = serviceClient.Call(grpc_metadata.NewOutgoingContext(ctx, grpc_metadata.Pairs(
		metadata.KeyRequestID, requestMsg.RequestID,
		metadata.KeyRecipientCMAccount, toCMAccount.Hex(),
		metadata.KeySenderCMAccount, fromCMAccount.Hex(),
	)), requestMsg.Content)
	if err != nil {
		return responseMsg, fmt.Errorf("error calling partner plugin service: %w", err)
	}

	if err := protovalidate.Validate(responseMsg.Content); err != nil {
		return responseMsg, fmt.Errorf("response message content validation failed: %w", err)
	}

	return responseMsg, nil
}

func (p *partnerPlugin) TokenBoughtNotificationWithoutBuyTx(ctx context.Context, tokenID *big.Int, mintID string) error {
	return p.tokenBoughtNotification(ctx, &notificationv3.TokenBought{
		TokenId: tokenID.Uint64(),
		MintId:  &typesv4.UUID{Value: mintID},
	})
}

func (p *partnerPlugin) TokenBoughtNotificationWithBuyTx(ctx context.Context, tokenID *big.Int, mintID string, buyTxID common.Hash) error {
	return p.tokenBoughtNotification(ctx, &notificationv3.TokenBought{
		TokenId: tokenID.Uint64(),
		MintId:  &typesv4.UUID{Value: mintID},
		TxId:    &typesv4.EVMTransactionID{Hash: buyTxID.Hex()},
	})
}

func (p *partnerPlugin) tokenBoughtNotification(ctx context.Context, notification *notificationv3.TokenBought) error {
	ctx, cancel := context.WithTimeout(ctx, p.responseTimeout)
	defer cancel()

	_, err := p.notificationClient.TokenBoughtNotification(ctx, notification)
	if err != nil {
		p.logger.Errorf("failed to send token bought notification: %v", err)
	}
	return err
}

func (p *partnerPlugin) TokenExpiredNotification(ctx context.Context, tokenID *big.Int, mintID string) error {
	ctx, cancel := context.WithTimeout(ctx, p.responseTimeout)
	defer cancel()

	_, err := p.notificationClient.TokenReservationExpiredNotification(ctx, &notificationv3.TokenReservationExpired{
		TokenId: tokenID.Uint64(),
		MintId:  &typesv4.UUID{Value: mintID},
	})
	if err != nil {
		p.logger.Errorf("failed to send token expired notification: %v", err)
	}
	return err
}

func (p *partnerPlugin) CancellationPendingNotification(ctx context.Context, notification *notificationv3.CancellationPending) error {
	ctx, cancel := context.WithTimeout(ctx, p.responseTimeout)
	defer cancel()

	_, err := p.notificationClient.CancellationPendingNotification(ctx, notification)
	if err != nil {
		p.logger.Errorf("failed to send cancellation pending notification: %v", err)
	}
	return err
}

func (p *partnerPlugin) CancellationRejectedNotification(ctx context.Context, notification *notificationv3.CancellationRejected) error {
	ctx, cancel := context.WithTimeout(ctx, p.responseTimeout)
	defer cancel()

	_, err := p.notificationClient.CancellationRejectedNotification(ctx, notification)
	if err != nil {
		p.logger.Errorf("failed to send cancellation rejected notification: %v", err)
	}
	return err
}

func (p *partnerPlugin) CancellationWithdrawnNotification(ctx context.Context, notification *notificationv3.CancellationWithdrawn) error {
	ctx, cancel := context.WithTimeout(ctx, p.responseTimeout)
	defer cancel()

	_, err := p.notificationClient.CancellationWithdrawnNotification(ctx, notification)
	if err != nil {
		p.logger.Errorf("failed to send cancellation withdrawn notification: %v", err)
	}
	return err
}

func (p *partnerPlugin) CancellationFinalizedNotification(ctx context.Context, notification *notificationv3.CancellationFinalized) error {
	ctx, cancel := context.WithTimeout(ctx, p.responseTimeout)
	defer cancel()

	_, err := p.notificationClient.CancellationFinalizedNotification(ctx, notification)
	if err != nil {
		p.logger.Errorf("failed to send cancellation finalized notification: %v", err)
	}
	return err
}
