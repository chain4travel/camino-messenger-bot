// Copyright (C) 2022-2025, Chain4Travel AG. All rights reserved.
// See the file LICENSE for licensing terms.

package v2

import (
	"context"
	"log"

	"buf.build/gen/go/chain4travel/camino-messenger-protocol/grpc/go/cmp/services/notification/v2/notificationv2grpc"
	notificationv2 "buf.build/gen/go/chain4travel/camino-messenger-protocol/protocolbuffers/go/cmp/services/notification/v2"
	"github.com/chain4travel/camino-messenger-bot/v11/pkg/metadata"
	"github.com/chain4travel/camino-messenger-bot/v11/pp-mock/events"
	"google.golang.org/protobuf/types/known/emptypb"
)

// Ensure that NotificationServiceV2Server implements the NotificationServiceServer interface
var _ notificationv2grpc.NotificationServiceServer = (*notificationServiceV2Server)(nil)

// NotificationServiceV2Server is the server that provides Notification services.
type notificationServiceV2Server struct {
	eventSender events.Sender
}

// NewNotificationServiceV2Server creates a new NotificationServiceV2Server.
func NewNotificationServiceV2Server(eventSender events.Sender) notificationv2grpc.NotificationServiceServer {
	return &notificationServiceV2Server{eventSender: eventSender}
}

// TokenBoughtNotification handles TokenBoughtNotification and returns a mock TokenBoughtNotificationResponse.
func (s *notificationServiceV2Server) TokenBoughtNotification(ctx context.Context, req *notificationv2.TokenBought) (*emptypb.Empty, error) {
	if err := s.eventSender.SendProtoEvent(req); err != nil {
		log.Printf("error sending event: %v", err)
	}

	md := metadata.FromGRPCContext(ctx)

	log.Printf("Responding to request: %s (TokenBoughtNotification)", md.RequestID)

	return &emptypb.Empty{}, nil
}

// TokenExpiredNotification handles TokenExpiredNotification and returns a mock TokenExpiredNotificationResponse.
func (s *notificationServiceV2Server) TokenExpiredNotification(ctx context.Context, req *notificationv2.TokenExpired) (*emptypb.Empty, error) {
	if err := s.eventSender.SendProtoEvent(req); err != nil {
		log.Printf("error sending event: %v", err)
	}

	md := metadata.FromGRPCContext(ctx)

	log.Printf("Responding to request: %s (TokenExpiredNotification)", md.RequestID)

	return &emptypb.Empty{}, nil
}

func (s *notificationServiceV2Server) CancellationPendingNotification(ctx context.Context, req *notificationv2.CancellationPending) (*emptypb.Empty, error) {
	if err := s.eventSender.SendProtoEvent(req); err != nil {
		log.Printf("error sending event: %v", err)
	}

	md := metadata.FromGRPCContext(ctx)

	log.Printf("Responding to request: %s (CancellationPendingNotification)", md.RequestID)

	return &emptypb.Empty{}, nil
}

func (s *notificationServiceV2Server) CancellationWithdrawnNotification(ctx context.Context, req *notificationv2.CancellationWithdrawn) (*emptypb.Empty, error) {
	if err := s.eventSender.SendProtoEvent(req); err != nil {
		log.Printf("error sending event: %v", err)
	}

	md := metadata.FromGRPCContext(ctx)

	log.Printf("Responding to request: %s (CancellationWithdrawnNotification)", md.RequestID)

	return &emptypb.Empty{}, nil
}

func (s *notificationServiceV2Server) CancellationRejectedNotification(ctx context.Context, req *notificationv2.CancellationRejected) (*emptypb.Empty, error) {
	if err := s.eventSender.SendProtoEvent(req); err != nil {
		log.Printf("error sending event: %v", err)
	}

	md := metadata.FromGRPCContext(ctx)

	log.Printf("Responding to request: %s (CancellationRejectedNotification)", md.RequestID)

	return &emptypb.Empty{}, nil
}

func (s *notificationServiceV2Server) CancellationFinalizedNotification(ctx context.Context, req *notificationv2.CancellationFinalized) (*emptypb.Empty, error) {
	if err := s.eventSender.SendProtoEvent(req); err != nil {
		log.Printf("error sending event: %v", err)
	}

	md := metadata.FromGRPCContext(ctx)

	log.Printf("Responding to request: %s (CancellationFinalizedNotification)", md.RequestID)

	return &emptypb.Empty{}, nil
}
