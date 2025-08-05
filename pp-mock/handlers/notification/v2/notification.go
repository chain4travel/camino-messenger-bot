// Copyright (C) 2022-2025, Chain4Travel AG. All rights reserved.
// See the file LICENSE for licensing terms.

package v2

import (
	"context"

	"buf.build/gen/go/chain4travel/camino-messenger-protocol/grpc/go/cmp/services/notification/v2/notificationv2grpc"
	notificationv2 "buf.build/gen/go/chain4travel/camino-messenger-protocol/protocolbuffers/go/cmp/services/notification/v2"
	"github.com/chain4travel/camino-messenger-bot/v11/pp-mock/handlers/state"
	"google.golang.org/protobuf/types/known/emptypb"
)

var _ notificationv2grpc.NotificationServiceServer = (*notificationServiceV2Server)(nil)

type notificationServiceV2Server struct{}

func NewNotificationServiceServer() notificationv2grpc.NotificationServiceServer {
	return &notificationServiceV2Server{}
}

func (s *notificationServiceV2Server) TokenBoughtNotification(_ context.Context, req *notificationv2.TokenBought) (*emptypb.Empty, error) {
	state.GetStore().SetMintBought(req.MintId.Value, true)
	return &emptypb.Empty{}, nil
}

func (s *notificationServiceV2Server) TokenExpiredNotification(_ context.Context, req *notificationv2.TokenExpired) (*emptypb.Empty, error) {
	state.GetStore().RemoveMintResult(req.MintId.Value)
	return &emptypb.Empty{}, nil
}

func (s *notificationServiceV2Server) CancellationPendingNotification(context.Context, *notificationv2.CancellationPending) (*emptypb.Empty, error) {
	return &emptypb.Empty{}, nil
}

func (s *notificationServiceV2Server) CancellationWithdrawnNotification(context.Context, *notificationv2.CancellationWithdrawn) (*emptypb.Empty, error) {
	return &emptypb.Empty{}, nil
}

func (s *notificationServiceV2Server) CancellationRejectedNotification(context.Context, *notificationv2.CancellationRejected) (*emptypb.Empty, error) {
	return &emptypb.Empty{}, nil
}

func (s *notificationServiceV2Server) CancellationFinalizedNotification(context.Context, *notificationv2.CancellationFinalized) (*emptypb.Empty, error) {
	return &emptypb.Empty{}, nil
}
