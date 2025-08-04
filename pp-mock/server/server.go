// Copyright (C) 2022-2025, Chain4Travel AG. All rights reserved.
// See the file LICENSE for licensing terms.

package server

import (
	"context"
	"fmt"
	"log"
	"net"
	"os"
	"os/signal"
	"strconv"
	"syscall"

	"github.com/chain4travel/camino-messenger-bot/v11/pp-mock/config"
	"github.com/chain4travel/camino-messenger-bot/v11/pp-mock/events"
	accommodation_v1 "github.com/chain4travel/camino-messenger-bot/v11/pp-mock/handlers/accommodation/v1"
	accommodation_v2 "github.com/chain4travel/camino-messenger-bot/v11/pp-mock/handlers/accommodation/v2"
	accommodation_v3 "github.com/chain4travel/camino-messenger-bot/v11/pp-mock/handlers/accommodation/v3"
	activity_v1 "github.com/chain4travel/camino-messenger-bot/v11/pp-mock/handlers/activity/v1"
	activity_v2 "github.com/chain4travel/camino-messenger-bot/v11/pp-mock/handlers/activity/v2"
	activity_v3 "github.com/chain4travel/camino-messenger-bot/v11/pp-mock/handlers/activity/v3"
	book_v1 "github.com/chain4travel/camino-messenger-bot/v11/pp-mock/handlers/book/v1"
	book_v2 "github.com/chain4travel/camino-messenger-bot/v11/pp-mock/handlers/book/v2"
	book_v3 "github.com/chain4travel/camino-messenger-bot/v11/pp-mock/handlers/book/v3"
	cancellation_v1 "github.com/chain4travel/camino-messenger-bot/v11/pp-mock/handlers/cancellation/v1"
	notification_v1 "github.com/chain4travel/camino-messenger-bot/v11/pp-mock/handlers/notification/v1"
	notification_v2 "github.com/chain4travel/camino-messenger-bot/v11/pp-mock/handlers/notification/v2"
	ping_v1 "github.com/chain4travel/camino-messenger-bot/v11/pp-mock/handlers/ping/v1"
	transport_v1 "github.com/chain4travel/camino-messenger-bot/v11/pp-mock/handlers/transport/v1"
	transport_v2 "github.com/chain4travel/camino-messenger-bot/v11/pp-mock/handlers/transport/v2"
	transport_v3 "github.com/chain4travel/camino-messenger-bot/v11/pp-mock/handlers/transport/v3"
	"github.com/chain4travel/camino-messenger-bot/v11/pp-mock/metadata"
	events_pb "github.com/chain4travel/camino-messenger-bot/v11/pp-mock/proto/pb/events"

	"buf.build/gen/go/chain4travel/camino-messenger-protocol/grpc/go/cmp/services/accommodation/v1/accommodationv1grpc"
	"buf.build/gen/go/chain4travel/camino-messenger-protocol/grpc/go/cmp/services/accommodation/v2/accommodationv2grpc"
	"buf.build/gen/go/chain4travel/camino-messenger-protocol/grpc/go/cmp/services/accommodation/v3/accommodationv3grpc"
	"buf.build/gen/go/chain4travel/camino-messenger-protocol/grpc/go/cmp/services/activity/v1/activityv1grpc"
	"buf.build/gen/go/chain4travel/camino-messenger-protocol/grpc/go/cmp/services/activity/v2/activityv2grpc"
	"buf.build/gen/go/chain4travel/camino-messenger-protocol/grpc/go/cmp/services/activity/v3/activityv3grpc"
	"buf.build/gen/go/chain4travel/camino-messenger-protocol/grpc/go/cmp/services/book/v1/bookv1grpc"
	"buf.build/gen/go/chain4travel/camino-messenger-protocol/grpc/go/cmp/services/book/v2/bookv2grpc"
	"buf.build/gen/go/chain4travel/camino-messenger-protocol/grpc/go/cmp/services/book/v3/bookv3grpc"
	"buf.build/gen/go/chain4travel/camino-messenger-protocol/grpc/go/cmp/services/cancellation/v1/cancellationv1grpc"
	"buf.build/gen/go/chain4travel/camino-messenger-protocol/grpc/go/cmp/services/notification/v1/notificationv1grpc"
	"buf.build/gen/go/chain4travel/camino-messenger-protocol/grpc/go/cmp/services/notification/v2/notificationv2grpc"
	"buf.build/gen/go/chain4travel/camino-messenger-protocol/grpc/go/cmp/services/ping/v1/pingv1grpc"
	"buf.build/gen/go/chain4travel/camino-messenger-protocol/grpc/go/cmp/services/transport/v1/transportv1grpc"
	"buf.build/gen/go/chain4travel/camino-messenger-protocol/grpc/go/cmp/services/transport/v2/transportv2grpc"
	"buf.build/gen/go/chain4travel/camino-messenger-protocol/grpc/go/cmp/services/transport/v3/transportv3grpc"
	"buf.build/go/protovalidate"

	"github.com/grpc-ecosystem/go-grpc-middleware/v2/interceptors"
	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"
	"google.golang.org/protobuf/proto"
)

const (
	EnvKeyEventsEnabled = "CMB_PARTNER_PLUGIN_MOCK_EVENTS"
	EnvKeyPort          = "CMB_PARTNER_PLUGIN_MOCK_PORT"
	EnvE2ETestMode      = "CMB_PARTNER_PLUGIN_MOCK_TEST_MODE"
	DefaultPort         = 50051
)

func Run() error {
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	config.SetDefaults()

	eventSender := events.NewDummySender()

	var eventServer events.Server
	if os.Getenv(EnvKeyEventsEnabled) == "true" {
		eventServer, eventSender = events.NewServer()
		eventServer.Start(ctx)
	}

	grpcServer := grpc.NewServer(
		grpc.ChainUnaryInterceptor(
			eventSenderInterceptor(eventSender),
			metadataExtractorInterceptor,
			loggingInterceptor,
			protoValidateInterceptor,
		),
	)

	events_pb.RegisterEventsServiceServer(grpcServer, eventServer)

	// Accommodation V1
	accommodationv1grpc.RegisterAccommodationSearchServiceServer(grpcServer, accommodation_v1.NewAccommodationSearchServer())
	accommodationv1grpc.RegisterAccommodationProductInfoServiceServer(grpcServer, accommodation_v1.NewAccommodationProductInfoServer())
	accommodationv1grpc.RegisterAccommodationProductListServiceServer(grpcServer, accommodation_v1.NewAccommodationProductListServer())
	// Accommodation V2
	accommodationv2grpc.RegisterAccommodationSearchServiceServer(grpcServer, accommodation_v2.NewAccommodationSearchServer())
	accommodationv2grpc.RegisterAccommodationProductInfoServiceServer(grpcServer, accommodation_v2.NewAccommodationProductInfoServer())
	accommodationv2grpc.RegisterAccommodationProductListServiceServer(grpcServer, accommodation_v2.NewAccommodationProductListServer())
	// Accommodation V3
	accommodationv3grpc.RegisterAccommodationSearchServiceServer(grpcServer, accommodation_v3.NewAccommodationSearchServer())
	accommodationv3grpc.RegisterAccommodationProductInfoServiceServer(grpcServer, accommodation_v3.NewAccommodationProductInfoServer())
	accommodationv3grpc.RegisterAccommodationProductListServiceServer(grpcServer, accommodation_v3.NewAccommodationProductListServer())

	// Activity V1
	activityv1grpc.RegisterActivityProductListServiceServer(grpcServer, activity_v1.NewActivityProductListServer())
	activityv1grpc.RegisterActivityProductInfoServiceServer(grpcServer, activity_v1.NewActivityProductInfoServer())
	activityv1grpc.RegisterActivitySearchServiceServer(grpcServer, activity_v1.NewActivitySearchServer())
	// Activity V2
	activityv2grpc.RegisterActivityProductListServiceServer(grpcServer, activity_v2.NewActivityProductListServer())
	activityv2grpc.RegisterActivityProductInfoServiceServer(grpcServer, activity_v2.NewActivityProductInfoServer())
	activityv2grpc.RegisterActivitySearchServiceServer(grpcServer, activity_v2.NewActivitySearchServer())
	// Activity V3
	activityv3grpc.RegisterActivityProductListServiceServer(grpcServer, activity_v3.NewActivityProductListServer())
	activityv3grpc.RegisterActivityProductInfoServiceServer(grpcServer, activity_v3.NewActivityProductInfoServer())
	activityv3grpc.RegisterActivitySearchServiceServer(grpcServer, activity_v3.NewActivitySearchServer())

	// Book V1
	bookv1grpc.RegisterMintServiceServer(grpcServer, book_v1.NewMintServiceServer())
	bookv1grpc.RegisterValidationServiceServer(grpcServer, book_v1.NewValidationServiceServer())
	// Book V2
	bookv2grpc.RegisterMintServiceServer(grpcServer, book_v2.NewMintServiceServer())
	bookv2grpc.RegisterValidationServiceServer(grpcServer, book_v2.NewValidationServiceServer())
	// Book V3
	bookv3grpc.RegisterMintServiceServer(grpcServer, book_v3.NewMintServiceServer())
	bookv3grpc.RegisterValidationServiceServer(grpcServer, book_v3.NewValidationServiceServer())

	// Ping V1
	pingv1grpc.RegisterPingServiceServer(grpcServer, ping_v1.NewPingServiceServer())

	// Notification V1
	notificationv1grpc.RegisterNotificationServiceServer(grpcServer, notification_v1.NewNotificationServiceServer())
	// Notification V2
	notificationv2grpc.RegisterNotificationServiceServer(grpcServer, notification_v2.NewNotificationServiceServer())

	// Transport V1
	transportv1grpc.RegisterTransportSearchServiceServer(grpcServer, transport_v1.NewTransportSearchServer())
	// Transport V2
	transportv2grpc.RegisterTransportSearchServiceServer(grpcServer, transport_v2.NewTransportSearchServer())
	// Transport V3
	transportv3grpc.RegisterTransportProductListServiceServer(grpcServer, transport_v3.NewTransportProductListServer())
	transportv3grpc.RegisterTransportSearchServiceServer(grpcServer, transport_v3.NewTransportSearchServer())

	// Cancellation V1
	cancellationv1grpc.RegisterCheckCancellationServiceServer(grpcServer, cancellation_v1.NewCheckCancellationServer())

	reflection.Register(grpcServer)

	port := DefaultPort
	var err error
	p, found := os.LookupEnv(EnvKeyPort)
	if found {
		port, err = strconv.Atoi(p)
		if err != nil {
			log.Printf("failed to parse port: %v", err)
			return err
		}
	}

	if os.Getenv(EnvE2ETestMode) == "true" {
		config.SetE2EDefaults()
	}

	log.SetOutput(os.Stdout)
	log.Printf("Starting server on port: %d", port)
	lis, err := net.Listen("tcp", fmt.Sprintf(":%d", port))
	if err != nil {
		log.Printf("failed to listen: %v", err)
		return err
	}

	go func() {
		<-ctx.Done()
		log.Printf("Shutting down server")
		grpcServer.Stop()
	}()

	if err := grpcServer.Serve(lis); err != nil {
		log.Printf("grpc server stopped serving: %v", err)
	}

	return nil
}

var (
	_ grpc.UnaryServerInterceptor = loggingInterceptor
	_ grpc.UnaryServerInterceptor = eventSenderInterceptor(nil)
	_ grpc.UnaryServerInterceptor = metadataExtractorInterceptor
)

func eventSenderInterceptor(eventSender events.Sender) grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req any, _ *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (any, error) {
		if protoReq, ok := req.(proto.Message); ok {
			if err := eventSender.SendProtoEvent(protoReq); err != nil {
				log.Printf("error sending event: %v", err)
			}
		}
		return handler(ctx, req)
	}
}

func metadataExtractorInterceptor(ctx context.Context, req any, _ *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (any, error) {
	return handler(metadata.ContextWithMetadata(ctx), req)
}

func loggingInterceptor(ctx context.Context, req any, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (any, error) {
	md := metadata.FromContext(ctx)
	callMeta := interceptors.NewServerCallMeta(info.FullMethod, nil, req)
	log.Printf("CMAccount %s received request from CMAccount %s", md.RecipientCMAccount, md.SenderCMAccount)
	log.Printf("Responding to %s: %s", callMeta.Service, md.RequestID)
	resp, err := handler(ctx, req)
	if err != nil {
		log.Printf("Error handling request: %v", err)
	}
	return resp, err
}

func protoValidateInterceptor(ctx context.Context, req any, _ *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (any, error) {
	protoReq, ok := req.(proto.Message)
	if !ok {
		return nil, fmt.Errorf("request is not a proto message: %T", req)
	}
	if err := protovalidate.Validate(protoReq); err != nil {
		return nil, fmt.Errorf("request validation failed: %w", err)
	}
	resp, err := handler(ctx, req)
	if err != nil {
		return resp, err
	}
	protoResp, ok := resp.(proto.Message)
	if !ok {
		return nil, fmt.Errorf("response is not a proto message: %T", resp)
	}
	if err := protovalidate.Validate(protoResp); err != nil {
		return nil, fmt.Errorf("response validation failed: %w", err)
	}
	return resp, nil
}
