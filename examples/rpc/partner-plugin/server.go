package main

import (
	"context"
	"fmt"
	"log"
	"net"
	"os"
	"strconv"

	"buf.build/gen/go/chain4travel/camino-messenger-protocol/grpc/go/cmp/services/accommodation/v1alpha/accommodationv1alphagrpc"
	"buf.build/gen/go/chain4travel/camino-messenger-protocol/grpc/go/cmp/services/activity/v1alpha/activityv1alphagrpc"
	"buf.build/gen/go/chain4travel/camino-messenger-protocol/grpc/go/cmp/services/network/v1alpha/networkv1alphagrpc"
	"buf.build/gen/go/chain4travel/camino-messenger-protocol/grpc/go/cmp/services/partner/v1alpha/partnerv1alphagrpc"
	"buf.build/gen/go/chain4travel/camino-messenger-protocol/grpc/go/cmp/services/ping/v1alpha/pingv1alphagrpc"
	"buf.build/gen/go/chain4travel/camino-messenger-protocol/grpc/go/cmp/services/transport/v1alpha/transportv1alphagrpc"

	accommodationv1alpha "buf.build/gen/go/chain4travel/camino-messenger-protocol/protocolbuffers/go/cmp/services/accommodation/v1alpha"
	activityv1alpha "buf.build/gen/go/chain4travel/camino-messenger-protocol/protocolbuffers/go/cmp/services/activity/v1alpha"
	networkv1alpha "buf.build/gen/go/chain4travel/camino-messenger-protocol/protocolbuffers/go/cmp/services/network/v1alpha"
	partnerv1alpha "buf.build/gen/go/chain4travel/camino-messenger-protocol/protocolbuffers/go/cmp/services/partner/v1alpha"
	pingv1alpha "buf.build/gen/go/chain4travel/camino-messenger-protocol/protocolbuffers/go/cmp/services/ping/v1alpha"
	transportv1alpha "buf.build/gen/go/chain4travel/camino-messenger-protocol/protocolbuffers/go/cmp/services/transport/v1alpha"
	typesv1alpha "buf.build/gen/go/chain4travel/camino-messenger-protocol/protocolbuffers/go/cmp/types/v1alpha"

	"github.com/chain4travel/camino-messenger-bot/internal/metadata"
	"google.golang.org/grpc"
)

type partnerPlugin struct {
	activityv1alphagrpc.ActivitySearchServiceServer
	accommodationv1alphagrpc.AccommodationSearchServiceServer
	networkv1alphagrpc.GetNetworkFeeServiceServer
	partnerv1alphagrpc.GetPartnerConfigurationServiceServer
	pingv1alphagrpc.PingServiceServer
	transportv1alphagrpc.TransportSearchServiceServer
}

func (p *partnerPlugin) ActivitySearch(ctx context.Context, request *activityv1alpha.ActivitySearchRequest) (*activityv1alpha.ActivitySearchResponse, error) {
	md := metadata.Metadata{}
	err := md.ExtractMetadata(ctx)
	if err != nil {
		log.Print("error extracting metadata")
	}
	md.Stamp(fmt.Sprintf("%s-%s", "ext-system", "response"))
	log.Printf("Responding to request: %s", md.RequestID)

	response := activityv1alpha.ActivitySearchResponse{
		Header:   nil,
		Metadata: &typesv1alpha.SearchResponseMetadata{SearchId: &typesv1alpha.UUID{Value: md.RequestID}},
	}
	grpc.SendHeader(ctx, md.ToGrpcMD())
	return &response, nil
}

func (p *partnerPlugin) AccommodationSearch(ctx context.Context, request *accommodationv1alpha.AccommodationSearchRequest) (*accommodationv1alpha.AccommodationSearchResponse, error) {
	md := metadata.Metadata{}
	err := md.ExtractMetadata(ctx)
	if err != nil {
		log.Print("error extracting metadata")
	}
	md.Stamp(fmt.Sprintf("%s-%s", "ext-system", "response"))
	log.Printf("Responding to request: %s", md.RequestID)

	response := accommodationv1alpha.AccommodationSearchResponse{
		Header:   nil,
		Metadata: &typesv1alpha.SearchResponseMetadata{SearchId: &typesv1alpha.UUID{Value: md.RequestID}},
	}
	grpc.SendHeader(ctx, md.ToGrpcMD())
	return &response, nil
}

func (p *partnerPlugin) GetNetworkFee(ctx context.Context, request *networkv1alpha.GetNetworkFeeRequest) (*networkv1alpha.GetNetworkFeeResponse, error) {
	md := metadata.Metadata{}
	err := md.ExtractMetadata(ctx)
	if err != nil {
		log.Print("error extracting metadata")
	}
	md.Stamp(fmt.Sprintf("%s-%s", "ext-system", "response"))
	log.Printf("Responding to request: %s", md.RequestID)

	response := networkv1alpha.GetNetworkFeeResponse{
		NetworkFee: &networkv1alpha.NetworkFee{
			Amount: 0,
			Asset:  nil,
		},
		CurrentBlockHeight: request.BlockHeight,
	}
	grpc.SendHeader(ctx, md.ToGrpcMD())
	return &response, nil
}
func (p *partnerPlugin) GetPartnerConfiguration(ctx context.Context, request *partnerv1alpha.GetPartnerConfigurationRequest) (*partnerv1alpha.GetPartnerConfigurationResponse, error) {
	md := metadata.Metadata{}
	err := md.ExtractMetadata(ctx)
	if err != nil {
		log.Print("error extracting metadata")
	}
	md.Stamp(fmt.Sprintf("%s-%s", "ext-system", "response"))
	log.Printf("Responding to request: %s", md.RequestID)

	response := partnerv1alpha.GetPartnerConfigurationResponse{
		PartnerConfiguration: &partnerv1alpha.PartnerConfiguration{},
		CurrentBlockHeight:   request.GetBlockHeight(),
	}
	grpc.SendHeader(ctx, md.ToGrpcMD())
	return &response, nil
}

func (p *partnerPlugin) Ping(ctx context.Context, request *pingv1alpha.PingRequest) (*pingv1alpha.PingResponse, error) {
	md := metadata.Metadata{}
	err := md.ExtractMetadata(ctx)
	if err != nil {
		log.Print("error extracting metadata")
	}
	md.Stamp(fmt.Sprintf("%s-%s", "ext-system", "response"))
	log.Printf("Responding to request: %s", md.RequestID)

	return &pingv1alpha.PingResponse{
		Header:      nil,
		PingMessage: fmt.Sprintf("Ping response to [%s] with request ID: %s", request.PingMessage, md.RequestID),
	}, nil
}
func (p *partnerPlugin) TransportSearch(ctx context.Context, request *transportv1alpha.TransportSearchRequest) (*transportv1alpha.TransportSearchResponse, error) {
	md := metadata.Metadata{}
	err := md.ExtractMetadata(ctx)
	if err != nil {
		log.Print("error extracting metadata")
	}
	md.Stamp(fmt.Sprintf("%s-%s", "ext-system", "response"))
	log.Printf("Responding to request: %s", md.RequestID)

	response := transportv1alpha.TransportSearchResponse{
		Header:   nil,
		Metadata: &typesv1alpha.SearchResponseMetadata{SearchId: &typesv1alpha.UUID{Value: md.RequestID}},
	}
	grpc.SendHeader(ctx, md.ToGrpcMD())
	return &response, nil
}

func main() {
	grpcServer := grpc.NewServer()
	activityv1alphagrpc.RegisterActivitySearchServiceServer(grpcServer, &partnerPlugin{})
	accommodationv1alphagrpc.RegisterAccommodationSearchServiceServer(grpcServer, &partnerPlugin{})
	networkv1alphagrpc.RegisterGetNetworkFeeServiceServer(grpcServer, &partnerPlugin{})
	partnerv1alphagrpc.RegisterGetPartnerConfigurationServiceServer(grpcServer, &partnerPlugin{})
	pingv1alphagrpc.RegisterPingServiceServer(grpcServer, &partnerPlugin{})
	transportv1alphagrpc.RegisterTransportSearchServiceServer(grpcServer, &partnerPlugin{})
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
	grpcServer.Serve(lis)
}
