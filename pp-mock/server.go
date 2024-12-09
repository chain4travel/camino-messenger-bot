package main

import (
	"fmt"
	"log"
	"net"
	"os"
	"strconv"

	"buf.build/gen/go/chain4travel/camino-messenger-protocol/grpc/go/cmp/services/accommodation/v1/accommodationv1grpc"
	"buf.build/gen/go/chain4travel/camino-messenger-protocol/grpc/go/cmp/services/accommodation/v2/accommodationv2grpc"
	"buf.build/gen/go/chain4travel/camino-messenger-protocol/grpc/go/cmp/services/book/v1/bookv1grpc"
	"buf.build/gen/go/chain4travel/camino-messenger-protocol/grpc/go/cmp/services/book/v2/bookv2grpc"
	"buf.build/gen/go/chain4travel/camino-messenger-protocol/grpc/go/cmp/services/ping/v1/pingv1grpc"
	handlers_accommodation_v1 "github.com/chain4travel/camino-messenger-bot/pp-mock/handlers/accommodation/v1"
	handlers_accommodation_v2 "github.com/chain4travel/camino-messenger-bot/pp-mock/handlers/accommodation/v2"
	handlers_mint_v1 "github.com/chain4travel/camino-messenger-bot/pp-mock/handlers/book/mint/v1"
	handlers_mint_v2 "github.com/chain4travel/camino-messenger-bot/pp-mock/handlers/book/mint/v2"
	handlers_validation_v1 "github.com/chain4travel/camino-messenger-bot/pp-mock/handlers/book/validation/v1"
	handlers_validation_v2 "github.com/chain4travel/camino-messenger-bot/pp-mock/handlers/book/validation/v2"
	handlers_ping_v1 "github.com/chain4travel/camino-messenger-bot/pp-mock/handlers/ping/v1"
	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"
)

func main() {
	grpcServer := grpc.NewServer()

	// Accommodation V1
	accommodationv1grpc.RegisterAccommodationSearchServiceServer(grpcServer, &handlers_accommodation_v1.AccommodationSearchV1Server{})
	accommodationv1grpc.RegisterAccommodationProductInfoServiceServer(grpcServer, &handlers_accommodation_v1.AccommodationProductInfoV1Server{})
	accommodationv1grpc.RegisterAccommodationProductListServiceServer(grpcServer, &handlers_accommodation_v1.AccommodationProductListV1Server{})

	// Accommodation V2
	accommodationv2grpc.RegisterAccommodationSearchServiceServer(grpcServer, &handlers_accommodation_v2.AccommodationSearchV2Server{})
	accommodationv2grpc.RegisterAccommodationProductInfoServiceServer(grpcServer, &handlers_accommodation_v2.AccommodationProductInfoV2Server{})
	accommodationv2grpc.RegisterAccommodationProductListServiceServer(grpcServer, &handlers_accommodation_v2.AccommodationProductListV2Server{})

	// Book - mint & validation
	// Book - Mint
	bookv2grpc.RegisterMintServiceServer(grpcServer, &handlers_mint_v2.MintServiceV2Server{})
	bookv1grpc.RegisterMintServiceServer(grpcServer, &handlers_mint_v1.MintServiceV1Server{})
	// Book - Validation
	bookv1grpc.RegisterValidationServiceServer(grpcServer, &handlers_validation_v1.ValidationServiceV1Server{})
	bookv2grpc.RegisterValidationServiceServer(grpcServer, &handlers_validation_v2.ValidationServiceV2Server{})

	// Ping
	pingv1grpc.RegisterPingServiceServer(grpcServer, &handlers_ping_v1.PingServiceV1Server{})

	reflection.Register(grpcServer)

	port := 55555
	var err error
	p, found := os.LookupEnv("PORT")
	if found {
		port, err = strconv.Atoi(p)
		if err != nil {
			panic(err)
		}
	}

	log.Printf("Starting server on port: %d", port)
	lis, err := net.Listen("tcp", fmt.Sprintf(":%d", port))
	if err != nil {
		log.Fatalf("failed to listen: %v", err)
	}

	if err := grpcServer.Serve(lis); err != nil {
		log.Fatalf("failed to serve: %v", err)
	}
}
