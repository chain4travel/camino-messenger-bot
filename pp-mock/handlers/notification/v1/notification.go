// Copyright (C) 2022-2025, Chain4Travel AG. All rights reserved.
// See the file LICENSE for licensing terms.

package v1

import (
	"context"

	"buf.build/gen/go/chain4travel/camino-messenger-protocol/grpc/go/cmp/services/notification/v1/notificationv1grpc"
	notificationv1 "buf.build/gen/go/chain4travel/camino-messenger-protocol/protocolbuffers/go/cmp/services/notification/v1"
	"github.com/chain4travel/camino-messenger-bot/v11/pp-mock/handlers/state"
	"google.golang.org/protobuf/types/known/emptypb"
)

var _ notificationv1grpc.NotificationServiceServer = (*notificationServiceV1Server)(nil)

type notificationServiceV1Server struct{}

func NewNotificationServiceServer() notificationv1grpc.NotificationServiceServer {
	return &notificationServiceV1Server{}
}

func (s *notificationServiceV1Server) TokenBoughtNotification(_ context.Context, req *notificationv1.TokenBought) (*emptypb.Empty, error) {
	state.GetStore().SetMintBought(req.MintId.Value, true)
	return &emptypb.Empty{}, nil
}

func (s *notificationServiceV1Server) TokenExpiredNotification(_ context.Context, req *notificationv1.TokenExpired) (*emptypb.Empty, error) {
	state.GetStore().RemoveMintResult(req.MintId.Value)
	return &emptypb.Empty{}, nil
}
