// Copyright (C) 2022-2025, Chain4Travel AG. All rights reserved.
// See the file LICENSE for licensing terms.

package partnerplugin

import (
	"context"
	"fmt"
	"math/big"
	"time"

	"buf.build/gen/go/chain4travel/camino-messenger-protocol/grpc/go/cmp/services/notification/v1/notificationv1grpc"
	notificationv1 "buf.build/gen/go/chain4travel/camino-messenger-protocol/protocolbuffers/go/cmp/services/notification/v1"
	typesv1 "buf.build/gen/go/chain4travel/camino-messenger-protocol/protocolbuffers/go/cmp/types/v1"
	"github.com/ethereum/go-ethereum/common"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
	"go.uber.org/zap"
	"google.golang.org/grpc"
	grpc_metadata "google.golang.org/grpc/metadata"

	types "github.com/chain4travel/camino-messenger-bot/v11/internal/messaging/types"
	rpc "github.com/chain4travel/camino-messenger-bot/v11/internal/rpc"
	"github.com/chain4travel/camino-messenger-bot/v11/internal/rpc/client"
)

var _ PartnerPlugin = (*partnerPlugin)(nil)

// Handles all communication with the partner plugin
type PartnerPlugin interface {
	DoServiceRequest(ctx context.Context, requestMsg *types.Message, service rpc.Client) (context.Context, *types.Message, error)
	SendTokenBoughtNotificationWithoutBuyTx(ctx context.Context, tokenID *big.Int, mintID string) error
	SendTokenBoughtNotificationWithBuyTx(ctx context.Context, tokenID *big.Int, mintID string, buyTxID common.Hash) error
	SendTokenExpiredNotification(ctx context.Context, tokenID *big.Int, mintID string) error
}

func New(
	logger *zap.SugaredLogger,
	tracer trace.Tracer,
	rpcClient *client.RPCClient,
	responseTimeout time.Duration,
) PartnerPlugin {
	return &partnerPlugin{
		logger:             logger,
		tracer:             tracer,
		notificationClient: notificationv1grpc.NewNotificationServiceClient(rpcClient.ClientConn),
		responseTimeout:    responseTimeout,
	}
}

type partnerPlugin struct {
	logger             *zap.SugaredLogger
	tracer             trace.Tracer
	notificationClient notificationv1grpc.NotificationServiceClient
	responseTimeout    time.Duration
}

func (p *partnerPlugin) DoServiceRequest(ctx context.Context, requestMsg *types.Message, service rpc.Client) (context.Context, *types.Message, error) {
	responseMsg := &types.Message{
		Metadata: requestMsg.Metadata,
	}

	var err error
	header := &grpc_metadata.MD{}
	ctx = grpc_metadata.NewOutgoingContext(ctx, requestMsg.Metadata.ToGrpcMD())
	ctx, partnerPluginSpan := p.tracer.Start(ctx, "service.Call", trace.WithSpanKind(trace.SpanKindClient), trace.WithAttributes(attribute.String("type", string(requestMsg.Type))))
	responseMsg.Content, responseMsg.Type, err = service.Call(ctx, requestMsg.Content, grpc.Header(header))
	partnerPluginSpan.End()
	if err != nil {
		return ctx, responseMsg, fmt.Errorf("error calling partner plugin service: %w", err)
	}

	if err := responseMsg.Metadata.FromGrpcMD(*header); err != nil {
		return ctx, responseMsg, fmt.Errorf("error extracting metadata from response: %w", err)
	}

	return ctx, responseMsg, nil
}

func (p *partnerPlugin) SendTokenBoughtNotificationWithoutBuyTx(ctx context.Context, tokenID *big.Int, mintID string) error {
	return p.sendTokenBoughtNotification(ctx, &notificationv1.TokenBought{
		TokenId: tokenID.Uint64(),
		MintId:  &typesv1.UUID{Value: mintID},
	})
}

func (p *partnerPlugin) SendTokenBoughtNotificationWithBuyTx(ctx context.Context, tokenID *big.Int, mintID string, buyTxID common.Hash) error {
	return p.sendTokenBoughtNotification(ctx, &notificationv1.TokenBought{
		TokenId: tokenID.Uint64(),
		MintId:  &typesv1.UUID{Value: mintID},
		TxId:    buyTxID.Hex(),
	})
}

func (p *partnerPlugin) sendTokenBoughtNotification(ctx context.Context, notification *notificationv1.TokenBought) error {
	ctx, cancel := context.WithTimeout(ctx, p.responseTimeout)
	defer cancel()

	_, err := p.notificationClient.TokenBoughtNotification(ctx, notification)
	if err != nil {
		p.logger.Errorf("failed to send token bought notification: %v", err)
	}
	return err
}

func (p *partnerPlugin) SendTokenExpiredNotification(ctx context.Context, tokenID *big.Int, mintID string) error {
	ctx, cancel := context.WithTimeout(ctx, p.responseTimeout)
	defer cancel()

	_, err := p.notificationClient.TokenExpiredNotification(ctx, &notificationv1.TokenExpired{
		TokenId: tokenID.Uint64(),
		MintId:  &typesv1.UUID{Value: mintID},
	})
	if err != nil {
		p.logger.Errorf("failed to send token expired notification: %v", err)
	}
	return err
}
