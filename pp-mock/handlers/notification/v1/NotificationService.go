package handlers_notification_v1

import (
	"context"
	"fmt"
	"log"

	"buf.build/gen/go/chain4travel/camino-messenger-protocol/grpc/go/cmp/services/notification/v1/notificationv1grpc"
	notificationv1 "buf.build/gen/go/chain4travel/camino-messenger-protocol/protocolbuffers/go/cmp/services/notification/v1"
	"github.com/chain4travel/camino-messenger-bot/internal/metadata"
	"google.golang.org/protobuf/types/known/emptypb"
)

// Ensure that NotificationServiceV1Server implements the NotificationServiceServer interface
var _ notificationv1grpc.NotificationServiceServer = (*NotificationServiceV1Server)(nil)

// NotificationServiceV1Server is the server that provides Notification services.
type NotificationServiceV1Server struct{}

// TokenBoughtNotification handles TokenBoughtNotification and returns a mock TokenBoughtNotificationResponse.
func (*NotificationServiceV1Server) TokenBoughtNotification(ctx context.Context, request *notificationv1.TokenBought) (*emptypb.Empty, error) {
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
func (*NotificationServiceV1Server) TokenExpiredNotification(ctx context.Context, request *notificationv1.TokenExpired) (*emptypb.Empty, error) {
	md := metadata.Metadata{}
	err := md.ExtractMetadata(ctx)
	if err != nil {
		log.Print("error extracting metadata")
	}
	md.Stamp(fmt.Sprintf("%s-%s", "ext-system", "response"))
	log.Printf("Responding to request: %s (TokenExpiredNotification)", md.RequestID)

	return &emptypb.Empty{}, nil
}
