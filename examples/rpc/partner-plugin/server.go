package main

import (
	"buf.build/gen/go/chain4travel/camino-messenger-protocol/grpc/go/cmp/services/ping/v1alpha1/pingv1alpha1grpc"
	pingv1alpha1 "buf.build/gen/go/chain4travel/camino-messenger-protocol/protocolbuffers/go/cmp/services/ping/v1alpha1"
	"context"
	"fmt"
	"log"
	"net"
	"os"
	"strconv"

	"buf.build/gen/go/chain4travel/camino-messenger-protocol/grpc/go/cmp/services/accommodation/v1alpha1/accommodationv1alpha1grpc"
	accommodationv1alpha1 "buf.build/gen/go/chain4travel/camino-messenger-protocol/protocolbuffers/go/cmp/services/accommodation/v1alpha1"
	"github.com/chain4travel/camino-messenger-bot/internal/metadata"
	"google.golang.org/grpc"
)

type partnerPlugin struct {
	accommodationv1alpha1grpc.AccommodationSearchServiceServer
	pingv1alpha1grpc.PingServiceServer
}

func (p *partnerPlugin) AccommodationSearch(ctx context.Context, request *accommodationv1alpha1.AccommodationSearchRequest) (*accommodationv1alpha1.AccommodationSearchResponse, error) {

	md := metadata.Metadata{}
	err := md.ExtractMetadata(ctx)
	if err != nil {
		log.Print("error extracting metadata")
	}
	md.Stamp(fmt.Sprintf("%s-%s", "ext-system", "response"))
	log.Printf("Responding to request: %s", md.RequestID)

	response := accommodationv1alpha1.AccommodationSearchResponse{
		Header:            nil,
		Context:           "",
		Errors:            "",
		Warnings:          "",
		SupplierCode:      "",
		ExternalSessionId: "",
		SearchId:          md.RequestID,
		Options:           nil,
	}
	grpc.SendHeader(ctx, md.ToGrpcMD())
	return &response, nil
}
func (p *partnerPlugin) Ping(ctx context.Context, request *pingv1alpha1.PingRequest) (*pingv1alpha1.PingResponse, error) {
	md := metadata.Metadata{}
	err := md.ExtractMetadata(ctx)
	if err != nil {
		log.Print("error extracting metadata")
	}
	md.Stamp(fmt.Sprintf("%s-%s", "ext-system", "response"))
	log.Printf("Responding to request: %s", md.RequestID)

	return &pingv1alpha1.PingResponse{
		Header:      nil,
		PingMessage: fmt.Sprintf("Ping response to [%s] with request ID: %s", request.PingMessage, md.RequestID),
	}, nil
}

func main() {
	grpcServer := grpc.NewServer()
	accommodationv1alpha1grpc.RegisterAccommodationSearchServiceServer(grpcServer, &partnerPlugin{})
	pingv1alpha1grpc.RegisterPingServiceServer(grpcServer, &partnerPlugin{})
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
	lis, err := net.Listen("tcp", fmt.Sprintf("localhost:%d", port))
	if err != nil {
		log.Fatalf("failed to listen: %v", err)
	}
	grpcServer.Serve(lis)
}
