// Copyright (C) 2022-2025, Chain4Travel AG. All rights reserved.
// See the file LICENSE for licensing terms.

package main

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
	"buf.build/gen/go/chain4travel/camino-messenger-protocol/grpc/go/cmp/services/ping/v1/pingv1grpc"
	"buf.build/gen/go/chain4travel/camino-messenger-protocol/grpc/go/cmp/services/transport/v1/transportv1grpc"
	"buf.build/gen/go/chain4travel/camino-messenger-protocol/grpc/go/cmp/services/transport/v2/transportv2grpc"
	"buf.build/gen/go/chain4travel/camino-messenger-protocol/grpc/go/cmp/services/transport/v3/transportv3grpc"

	handlers_accommodation_v1 "github.com/chain4travel/camino-messenger-bot/pp-mock/handlers/accommodation/v1"
	handlers_accommodation_v2 "github.com/chain4travel/camino-messenger-bot/pp-mock/handlers/accommodation/v2"
	handlers_accommodation_v3 "github.com/chain4travel/camino-messenger-bot/pp-mock/handlers/accommodation/v3"
	handlers_mint_v1 "github.com/chain4travel/camino-messenger-bot/pp-mock/handlers/book/mint/v1"
	handlers_mint_v2 "github.com/chain4travel/camino-messenger-bot/pp-mock/handlers/book/mint/v2"
	handlers_validation_v1 "github.com/chain4travel/camino-messenger-bot/pp-mock/handlers/book/validation/v1"
	handlers_validation_v2 "github.com/chain4travel/camino-messenger-bot/pp-mock/handlers/book/validation/v2"
	handlers_ping_v1 "github.com/chain4travel/camino-messenger-bot/pp-mock/handlers/ping/v1"
	handlers_transport_v1 "github.com/chain4travel/camino-messenger-bot/pp-mock/handlers/transport/v1"
	handlers_transport_v2 "github.com/chain4travel/camino-messenger-bot/pp-mock/handlers/transport/v2"
	handlers_transport_v3 "github.com/chain4travel/camino-messenger-bot/pp-mock/handlers/transport/v3"
)

func main() {
	if err := run(); err != nil {
		os.Exit(1)
	}
}

func run() error {
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	grpcServer := grpc.NewServer()

	// Accommodation V1
	accommodationv1grpc.RegisterAccommodationSearchServiceServer(grpcServer, &handlers_accommodation_v1.AccommodationSearchV1Server{})
	accommodationv1grpc.RegisterAccommodationProductInfoServiceServer(grpcServer, &handlers_accommodation_v1.AccommodationProductInfoV1Server{})
	accommodationv1grpc.RegisterAccommodationProductListServiceServer(grpcServer, &handlers_accommodation_v1.AccommodationProductListV1Server{})

	// Accommodation V2
	accommodationv2grpc.RegisterAccommodationSearchServiceServer(grpcServer, &handlers_accommodation_v2.AccommodationSearchV2Server{})
	accommodationv2grpc.RegisterAccommodationProductInfoServiceServer(grpcServer, &handlers_accommodation_v2.AccommodationProductInfoV2Server{})
	accommodationv2grpc.RegisterAccommodationProductListServiceServer(grpcServer, &handlers_accommodation_v2.AccommodationProductListV2Server{})

	// Accommodation V3
	accommodationv3grpc.RegisterAccommodationSearchServiceServer(grpcServer, &handlers_accommodation_v3.AccommodationSearchV3Server{})
	accommodationv3grpc.RegisterAccommodationProductInfoServiceServer(grpcServer, &handlers_accommodation_v3.AccommodationProductInfoV3Server{})
	accommodationv3grpc.RegisterAccommodationProductListServiceServer(grpcServer, &handlers_accommodation_v3.AccommodationProductListV3Server{})

	// Book - mint & validation
	// Book - Mint
	bookv2grpc.RegisterMintServiceServer(grpcServer, &handlers_mint_v2.MintServiceV2Server{})
	bookv1grpc.RegisterMintServiceServer(grpcServer, &handlers_mint_v1.MintServiceV1Server{})
	// Book - Validation
	bookv1grpc.RegisterValidationServiceServer(grpcServer, &handlers_validation_v1.ValidationServiceV1Server{})
	bookv2grpc.RegisterValidationServiceServer(grpcServer, &handlers_validation_v2.ValidationServiceV2Server{})

	// Ping
	pingv1grpc.RegisterPingServiceServer(grpcServer, &handlers_ping_v1.PingServiceV1Server{})

	// Transport
	transportv1grpc.RegisterTransportSearchServiceServer(grpcServer, &handlers_transport_v1.TransportSearchV1Server{})

	transportv2grpc.RegisterTransportSearchServiceServer(grpcServer, &handlers_transport_v2.TransportSearchV2Server{})

	transportv3grpc.RegisterTransportProductListServiceServer(grpcServer, &handlers_transport_v3.TransportProductListV3Server{})
	transportv3grpc.RegisterTransportSearchServiceServer(grpcServer, &handlers_transport_v3.TransportSearchV3Server{})

	reflection.Register(grpcServer)

	port := 55555
	var err error
	p, found := os.LookupEnv("CMB_PARTNER_PLUGIN_MOCK_PORT")
	if found {
		port, err = strconv.Atoi(p)
		if err != nil {
			log.Printf("failed to parse port: %v", err)
			return err
		}
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
