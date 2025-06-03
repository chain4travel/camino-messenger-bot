// Copyright (C) 2022-2025, Chain4Travel AG. All rights reserved.
// See the file LICENSE for licensing terms.

package v1

import (
	"context"
	"fmt"
	"log"

	"buf.build/gen/go/chain4travel/camino-messenger-protocol/grpc/go/cmp/services/notification/v1/notificationv1grpc"
	notificationv1 "buf.build/gen/go/chain4travel/camino-messenger-protocol/protocolbuffers/go/cmp/services/notification/v1"
	"github.com/chain4travel/camino-messenger-bot/v11/pkg/metadata"
	"github.com/chain4travel/camino-messenger-bot/v11/pp-mock/events"
	"google.golang.org/protobuf/types/known/emptypb"
)

// Ensure that NotificationServiceV1Server implements the NotificationServiceServer interface
var _ notificationv1grpc.NotificationServiceServer = (*notificationServiceV1Server)(nil)

// NotificationServiceV1Server is the server that provides Notification services.
type notificationServiceV1Server struct {
	eventSender events.Sender
}

// NewNotificationServiceV1Server creates a new NotificationServiceV1Server.
func NewNotificationServiceV1Server(eventSender events.Sender) notificationv1grpc.NotificationServiceServer {
	return &notificationServiceV1Server{eventSender: eventSender}
}

// TokenBoughtNotification handles TokenBoughtNotification and returns a mock TokenBoughtNotificationResponse.
func (s *notificationServiceV1Server) TokenBoughtNotification(ctx context.Context, req *notificationv1.TokenBought) (*emptypb.Empty, error) {
	if err := s.eventSender.SendProtoEvent(req); err != nil {
		log.Printf("error sending event: %v", err)
	}

	md := metadata.Metadata{}
	err := md.ExtractMetadata(ctx)
	if err != nil {
		log.Print("error extracting metadata")
	}
	md.Stamp(fmt.Sprintf("%s-%s", "ext-system", "response"))
	log.Printf("Responding to request: %s (TokenBoughtNotification)", md.RequestID)

	return &emptypb.Empty{}, nil
}

// TokenExpiredNotification handles TokenExpiredNotification and returns a mock TokenExpiredNotificationResponse.
func (s *notificationServiceV1Server) TokenExpiredNotification(ctx context.Context, req *notificationv1.TokenExpired) (*emptypb.Empty, error) {
	if err := s.eventSender.SendProtoEvent(req); err != nil {
		log.Printf("error sending event: %v", err)
	}

	md := metadata.Metadata{}
	err := md.ExtractMetadata(ctx)
	if err != nil {
		log.Print("error extracting metadata")
	}
	md.Stamp(fmt.Sprintf("%s-%s", "ext-system", "response"))
	log.Printf("Responding to request: %s (TokenExpiredNotification)", md.RequestID)

	return &emptypb.Empty{}, nil
}
