// Copyright (C) 2022-2026, Chain4Travel AG. All rights reserved.
// See the file LICENSE for licensing terms.

package v3

import (
	"context"

	"buf.build/gen/go/chain4travel/camino-messenger-protocol/grpc/go/cmp/services/notification/v3/notificationv3grpc"
	notificationv3 "buf.build/gen/go/chain4travel/camino-messenger-protocol/protocolbuffers/go/cmp/services/notification/v3"
	"github.com/chain4travel/camino-messenger-bot/v12/pp-mock/handlers/state"
	"google.golang.org/protobuf/types/known/emptypb"
)

var _ notificationv3grpc.NotificationServiceServer = (*notificationServiceV3Server)(nil)

type notificationServiceV3Server struct{}

func NewNotificationServiceServer() notificationv3grpc.NotificationServiceServer {
	return &notificationServiceV3Server{}
}

func (s *notificationServiceV3Server) TokenBoughtNotification(_ context.Context, req *notificationv3.TokenBought) (*emptypb.Empty, error) {
	state.GetStore().SetMintBought(req.MintId.Value, true)
	return &emptypb.Empty{}, nil
}

func (s *notificationServiceV3Server) TokenReservationExpiredNotification(_ context.Context, req *notificationv3.TokenReservationExpired) (*emptypb.Empty, error) {
	state.GetStore().RemoveMintResult(req.MintId.Value)
	return &emptypb.Empty{}, nil
}

func (s *notificationServiceV3Server) CancellationPendingNotification(context.Context, *notificationv3.CancellationPending) (*emptypb.Empty, error) {
	return &emptypb.Empty{}, nil
}

func (s *notificationServiceV3Server) CancellationWithdrawnNotification(context.Context, *notificationv3.CancellationWithdrawn) (*emptypb.Empty, error) {
	return &emptypb.Empty{}, nil
}

func (s *notificationServiceV3Server) CancellationRejectedNotification(context.Context, *notificationv3.CancellationRejected) (*emptypb.Empty, error) {
	return &emptypb.Empty{}, nil
}

func (s *notificationServiceV3Server) CancellationFinalizedNotification(context.Context, *notificationv3.CancellationFinalized) (*emptypb.Empty, error) {
	return &emptypb.Empty{}, nil
}
