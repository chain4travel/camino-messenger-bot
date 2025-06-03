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

	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"

	"buf.build/gen/go/chain4travel/camino-messenger-protocol/grpc/go/cmp/services/accommodation/v1/accommodationv1grpc"
	"buf.build/gen/go/chain4travel/camino-messenger-protocol/grpc/go/cmp/services/accommodation/v2/accommodationv2grpc"
	"buf.build/gen/go/chain4travel/camino-messenger-protocol/grpc/go/cmp/services/accommodation/v3/accommodationv3grpc"
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

	"github.com/chain4travel/camino-messenger-bot/v11/pp-mock/config"
	"github.com/chain4travel/camino-messenger-bot/v11/pp-mock/events"
	accommodation_v1 "github.com/chain4travel/camino-messenger-bot/v11/pp-mock/handlers/accommodation/v1"
	accommodation_v2 "github.com/chain4travel/camino-messenger-bot/v11/pp-mock/handlers/accommodation/v2"
	accommodation_v3 "github.com/chain4travel/camino-messenger-bot/v11/pp-mock/handlers/accommodation/v3"
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
	events_pb "github.com/chain4travel/camino-messenger-bot/v11/pp-mock/proto/pb/events"
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

	grpcServer := grpc.NewServer()

	eventSender := events.NewDummySender()
	if os.Getenv(EnvKeyEventsEnabled) == "true" {
		var eventServer events.Server
		eventServer, eventSender = events.NewServer()
		eventServer.Start(ctx)
		events_pb.RegisterEventsServiceServer(grpcServer, eventServer)
	}

	// Accommodation V1
	accommodationv1grpc.RegisterAccommodationSearchServiceServer(grpcServer, accommodation_v1.NewAccommodationSearchV1Server(eventSender))
	accommodationv1grpc.RegisterAccommodationProductInfoServiceServer(grpcServer, accommodation_v1.NewAccommodationProductInfoV1Server(eventSender))
	accommodationv1grpc.RegisterAccommodationProductListServiceServer(grpcServer, accommodation_v1.NewAccommodationProductListV1Server(eventSender))
	// Accommodation V2
	accommodationv2grpc.RegisterAccommodationSearchServiceServer(grpcServer, accommodation_v2.NewAccommodationSearchV2Server(eventSender))
	accommodationv2grpc.RegisterAccommodationProductInfoServiceServer(grpcServer, accommodation_v2.NewAccommodationProductInfoV2Server(eventSender))
	accommodationv2grpc.RegisterAccommodationProductListServiceServer(grpcServer, accommodation_v2.NewAccommodationProductListV2Server(eventSender))
	// Accommodation V3
	accommodationv3grpc.RegisterAccommodationSearchServiceServer(grpcServer, accommodation_v3.NewAccommodationSearchV3Server(eventSender))
	accommodationv3grpc.RegisterAccommodationProductInfoServiceServer(grpcServer, accommodation_v3.NewAccommodationProductInfoV3Server(eventSender))
	accommodationv3grpc.RegisterAccommodationProductListServiceServer(grpcServer, accommodation_v3.NewAccommodationProductListV3Server(eventSender))

	// Book V1
	bookv1grpc.RegisterMintServiceServer(grpcServer, book_v1.NewMintServiceV1Server(eventSender))
	bookv1grpc.RegisterValidationServiceServer(grpcServer, book_v1.NewValidationServiceV1Server(eventSender))
	// Book V2
	bookv2grpc.RegisterMintServiceServer(grpcServer, book_v2.NewMintServiceV2Server(eventSender))
	bookv2grpc.RegisterValidationServiceServer(grpcServer, book_v2.NewValidationServiceV2Server(eventSender))
	// Book V3
	bookv3grpc.RegisterMintServiceServer(grpcServer, book_v3.NewMintServiceV3Server(eventSender))
	bookv3grpc.RegisterValidationServiceServer(grpcServer, book_v3.NewValidationServiceV3Server(eventSender))

	// Ping V1
	pingv1grpc.RegisterPingServiceServer(grpcServer, ping_v1.NewPingServiceV1Server(eventSender))

	// Notification V1
	notificationv1grpc.RegisterNotificationServiceServer(grpcServer, notification_v1.NewNotificationServiceV1Server(eventSender))
	// Notification V2
	notificationv2grpc.RegisterNotificationServiceServer(grpcServer, notification_v2.NewNotificationServiceV2Server(eventSender))

	// Transport V1
	transportv1grpc.RegisterTransportSearchServiceServer(grpcServer, transport_v1.NewTransportSearchV1Server(eventSender))
	// Transport V2
	transportv2grpc.RegisterTransportSearchServiceServer(grpcServer, transport_v2.NewTransportSearchV2Server(eventSender))
	// Transport V3
	transportv3grpc.RegisterTransportProductListServiceServer(grpcServer, transport_v3.NewTransportProductListV3Server(eventSender))
	transportv3grpc.RegisterTransportSearchServiceServer(grpcServer, transport_v3.NewTransportSearchV3Server(eventSender))

	// Cancellation V1
	cancellationv1grpc.RegisterCheckCancellationServiceServer(grpcServer, cancellation_v1.NewCheckCancellationV1Server(eventSender))

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
