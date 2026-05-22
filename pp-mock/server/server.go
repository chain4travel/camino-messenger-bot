// Copyright (C) 2022-2026, Chain4Travel AG. All rights reserved.
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

	"github.com/chain4travel/camino-messenger-bot/v13/pp-mock/config"
	"github.com/chain4travel/camino-messenger-bot/v13/pp-mock/events"
	accommodation_v3 "github.com/chain4travel/camino-messenger-bot/v13/pp-mock/handlers/accommodation/v3"
	accommodation_v4 "github.com/chain4travel/camino-messenger-bot/v13/pp-mock/handlers/accommodation/v4"
	accommodation_v5 "github.com/chain4travel/camino-messenger-bot/v13/pp-mock/handlers/accommodation/v5"
	activity_v3 "github.com/chain4travel/camino-messenger-bot/v13/pp-mock/handlers/activity/v3"
	activity_v4 "github.com/chain4travel/camino-messenger-bot/v13/pp-mock/handlers/activity/v4"
	activity_v5 "github.com/chain4travel/camino-messenger-bot/v13/pp-mock/handlers/activity/v5"
	book_v3 "github.com/chain4travel/camino-messenger-bot/v13/pp-mock/handlers/book/v3"
	book_v4 "github.com/chain4travel/camino-messenger-bot/v13/pp-mock/handlers/book/v4"
	book_v5 "github.com/chain4travel/camino-messenger-bot/v13/pp-mock/handlers/book/v5"
	cancellation_v1 "github.com/chain4travel/camino-messenger-bot/v13/pp-mock/handlers/cancellation/v1"
	cancellation_v2 "github.com/chain4travel/camino-messenger-bot/v13/pp-mock/handlers/cancellation/v2"
	cancellation_v3 "github.com/chain4travel/camino-messenger-bot/v13/pp-mock/handlers/cancellation/v3"
	notification_v1 "github.com/chain4travel/camino-messenger-bot/v13/pp-mock/handlers/notification/v1"
	notification_v2 "github.com/chain4travel/camino-messenger-bot/v13/pp-mock/handlers/notification/v2"
	notification_v3 "github.com/chain4travel/camino-messenger-bot/v13/pp-mock/handlers/notification/v3"
	ping_v1 "github.com/chain4travel/camino-messenger-bot/v13/pp-mock/handlers/ping/v1"
	ping_v2 "github.com/chain4travel/camino-messenger-bot/v13/pp-mock/handlers/ping/v2"
	seat_map_v4 "github.com/chain4travel/camino-messenger-bot/v13/pp-mock/handlers/seat_map/v4"
	transport_v3 "github.com/chain4travel/camino-messenger-bot/v13/pp-mock/handlers/transport/v3"
	transport_v4 "github.com/chain4travel/camino-messenger-bot/v13/pp-mock/handlers/transport/v4"
	transport_v5 "github.com/chain4travel/camino-messenger-bot/v13/pp-mock/handlers/transport/v5"
	"github.com/chain4travel/camino-messenger-bot/v13/pp-mock/metadata"
	events_pb "github.com/chain4travel/camino-messenger-bot/v13/pp-mock/proto/pb/events"

	"buf.build/gen/go/chain4travel/camino-messenger-protocol/grpc/go/cmp/services/accommodation/v3/accommodationv3grpc"
	"buf.build/gen/go/chain4travel/camino-messenger-protocol/grpc/go/cmp/services/accommodation/v4/accommodationv4grpc"
	"buf.build/gen/go/chain4travel/camino-messenger-protocol/grpc/go/cmp/services/accommodation/v5/accommodationv5grpc"
	"buf.build/gen/go/chain4travel/camino-messenger-protocol/grpc/go/cmp/services/activity/v3/activityv3grpc"
	"buf.build/gen/go/chain4travel/camino-messenger-protocol/grpc/go/cmp/services/activity/v4/activityv4grpc"
	"buf.build/gen/go/chain4travel/camino-messenger-protocol/grpc/go/cmp/services/activity/v5/activityv5grpc"
	"buf.build/gen/go/chain4travel/camino-messenger-protocol/grpc/go/cmp/services/book/v3/bookv3grpc"
	"buf.build/gen/go/chain4travel/camino-messenger-protocol/grpc/go/cmp/services/book/v4/bookv4grpc"
	"buf.build/gen/go/chain4travel/camino-messenger-protocol/grpc/go/cmp/services/book/v5/bookv5grpc"
	"buf.build/gen/go/chain4travel/camino-messenger-protocol/grpc/go/cmp/services/cancellation/v1/cancellationv1grpc"
	"buf.build/gen/go/chain4travel/camino-messenger-protocol/grpc/go/cmp/services/cancellation/v2/cancellationv2grpc"
	"buf.build/gen/go/chain4travel/camino-messenger-protocol/grpc/go/cmp/services/cancellation/v3/cancellationv3grpc"
	"buf.build/gen/go/chain4travel/camino-messenger-protocol/grpc/go/cmp/services/notification/v1/notificationv1grpc"
	"buf.build/gen/go/chain4travel/camino-messenger-protocol/grpc/go/cmp/services/notification/v2/notificationv2grpc"
	"buf.build/gen/go/chain4travel/camino-messenger-protocol/grpc/go/cmp/services/notification/v3/notificationv3grpc"
	"buf.build/gen/go/chain4travel/camino-messenger-protocol/grpc/go/cmp/services/ping/v1/pingv1grpc"
	"buf.build/gen/go/chain4travel/camino-messenger-protocol/grpc/go/cmp/services/ping/v2/pingv2grpc"
	"buf.build/gen/go/chain4travel/camino-messenger-protocol/grpc/go/cmp/services/seat_map/v4/seat_mapv4grpc"
	"buf.build/gen/go/chain4travel/camino-messenger-protocol/grpc/go/cmp/services/transport/v3/transportv3grpc"
	"buf.build/gen/go/chain4travel/camino-messenger-protocol/grpc/go/cmp/services/transport/v4/transportv4grpc"
	"buf.build/gen/go/chain4travel/camino-messenger-protocol/grpc/go/cmp/services/transport/v5/transportv5grpc"
	"buf.build/go/protovalidate"

	"github.com/grpc-ecosystem/go-grpc-middleware/v2/interceptors"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/reflection"
	"google.golang.org/grpc/status"
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
	// Accommodation V3
	accommodationv3grpc.RegisterAccommodationSearchServiceServer(grpcServer, accommodation_v3.NewAccommodationSearchServer())
	accommodationv3grpc.RegisterAccommodationProductInfoServiceServer(grpcServer, accommodation_v3.NewAccommodationProductInfoServer())
	accommodationv3grpc.RegisterAccommodationProductListServiceServer(grpcServer, accommodation_v3.NewAccommodationProductListServer())
	// Accommodation V4
	accommodationv4grpc.RegisterAccommodationSearchServiceServer(grpcServer, accommodation_v4.NewAccommodationSearchServer())
	accommodationv4grpc.RegisterAccommodationProductInfoServiceServer(grpcServer, accommodation_v4.NewAccommodationProductInfoServer())
	accommodationv4grpc.RegisterAccommodationProductListServiceServer(grpcServer, accommodation_v4.NewAccommodationProductListServer())
	accommodationv4grpc.RegisterAccommodationProductShortListServiceServer(grpcServer, accommodation_v4.NewAccommodationProductShortListServer())
	// Accommodation V5
	accommodationv5grpc.RegisterAccommodationSearchServiceServer(grpcServer, accommodation_v5.NewAccommodationSearchServer())
	accommodationv5grpc.RegisterAccommodationProductInfoServiceServer(grpcServer, accommodation_v5.NewAccommodationProductInfoServer())
	accommodationv5grpc.RegisterAccommodationProductListServiceServer(grpcServer, accommodation_v5.NewAccommodationProductListServer())
	accommodationv5grpc.RegisterAccommodationProductShortListServiceServer(grpcServer, accommodation_v5.NewAccommodationProductShortListServer())

	// Activity V3
	activityv3grpc.RegisterActivityProductListServiceServer(grpcServer, activity_v3.NewActivityProductListServer())
	activityv3grpc.RegisterActivityProductInfoServiceServer(grpcServer, activity_v3.NewActivityProductInfoServer())
	activityv3grpc.RegisterActivitySearchServiceServer(grpcServer, activity_v3.NewActivitySearchServer())
	// Activity V4
	activityv4grpc.RegisterActivityProductListServiceServer(grpcServer, activity_v4.NewActivityProductListServer())
	activityv4grpc.RegisterActivityProductInfoServiceServer(grpcServer, activity_v4.NewActivityProductInfoServer())
	activityv4grpc.RegisterActivitySearchServiceServer(grpcServer, activity_v4.NewActivitySearchServer())
	activityv4grpc.RegisterActivityProductShortListServiceServer(grpcServer, activity_v4.NewActivityProductShortListServer())
	// Activity V5
	activityv5grpc.RegisterActivitySearchServiceServer(grpcServer, activity_v5.NewActivitySearchServer())
	activityv5grpc.RegisterActivityProductListServiceServer(grpcServer, activity_v5.NewActivityProductListServer())
	activityv5grpc.RegisterActivityProductInfoServiceServer(grpcServer, activity_v5.NewActivityProductInfoServer())
	activityv5grpc.RegisterActivityProductShortListServiceServer(grpcServer, activity_v5.NewActivityProductShortListServer())

	// Book V3
	bookv3grpc.RegisterMintServiceServer(grpcServer, book_v3.NewMintServiceServer())
	bookv3grpc.RegisterValidationServiceServer(grpcServer, book_v3.NewValidationServiceServer())
	// Book V4
	bookv4grpc.RegisterMintServiceServer(grpcServer, book_v4.NewMintServiceServer())
	bookv4grpc.RegisterValidationServiceServer(grpcServer, book_v4.NewValidationServiceServer())
	// Book V5
	bookv5grpc.RegisterMintServiceServer(grpcServer, book_v5.NewMintServiceServer())
	bookv5grpc.RegisterValidationServiceServer(grpcServer, book_v5.NewValidationServiceServer())

	// Ping V1
	pingv1grpc.RegisterPingServiceServer(grpcServer, ping_v1.NewPingServiceServer())
	// Ping V2
	pingv2grpc.RegisterPingServiceServer(grpcServer, ping_v2.NewPingServiceServer())

	// Notification V1
	notificationv1grpc.RegisterNotificationServiceServer(grpcServer, notification_v1.NewNotificationServiceServer())
	// Notification V2
	notificationv2grpc.RegisterNotificationServiceServer(grpcServer, notification_v2.NewNotificationServiceServer())
	// Notification V3
	notificationv3grpc.RegisterNotificationServiceServer(grpcServer, notification_v3.NewNotificationServiceServer())

	// Transport V3
	transportv3grpc.RegisterTransportProductListServiceServer(grpcServer, transport_v3.NewTransportProductListServer())
	transportv3grpc.RegisterTransportSearchServiceServer(grpcServer, transport_v3.NewTransportSearchServer())
	// Transport V4
	transportv4grpc.RegisterTransportProductListServiceServer(grpcServer, transport_v4.NewTransportProductListServer())
	transportv4grpc.RegisterTransportSearchServiceServer(grpcServer, transport_v4.NewTransportSearchServer())
	// Transport V5
	transportv5grpc.RegisterTransportProductListServiceServer(grpcServer, transport_v5.NewTransportProductListServer())
	transportv5grpc.RegisterTransportSearchServiceServer(grpcServer, transport_v5.NewTransportSearchServer())

	// SeatMap V4
	seat_mapv4grpc.RegisterSeatMapServiceServer(grpcServer, seat_map_v4.NewSeatMapServer())
	seat_mapv4grpc.RegisterSeatMapAvailabilityServiceServer(grpcServer, seat_map_v4.NewSeatMapAvailabilityServer())

	// Cancellation V1
	cancellationv1grpc.RegisterCheckCancellationServiceServer(grpcServer, cancellation_v1.NewCheckCancellationServer())
	// Cancellation V2
	cancellationv2grpc.RegisterCheckCancellationServiceServer(grpcServer, cancellation_v2.NewCheckCancellationServer())
	// Cancellation V3
	cancellationv3grpc.RegisterCheckCancellationServiceServer(grpcServer, cancellation_v3.NewCheckCancellationServer())

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
	listenCfg := net.ListenConfig{}
	lis, err := listenCfg.Listen(ctx, "tcp", fmt.Sprintf(":%d", port))
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
		return resp, status.Errorf(codes.Internal, "error handling request: %v", err)
	}
	return resp, nil
}

func protoValidateInterceptor(ctx context.Context, req any, _ *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (any, error) {
	protoReq, ok := req.(proto.Message)
	if !ok {
		return nil, status.Errorf(codes.InvalidArgument, "request is not a proto message: %T", req)
	}
	if err := protovalidate.Validate(protoReq); err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "request validation failed: %v", err)
	}
	resp, err := handler(ctx, req)
	if err != nil {
		return resp, err
	}
	protoResp, ok := resp.(proto.Message)
	if !ok {
		return nil, status.Errorf(codes.Internal, "response is not a proto message: %T", resp)
	}
	if err := protovalidate.Validate(protoResp); err != nil {
		return nil, status.Errorf(codes.Internal, "response validation failed: %v", err)
	}
	return resp, nil
}
