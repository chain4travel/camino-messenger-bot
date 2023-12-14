package main

import (
	"context"
	"fmt"
	"log"
	"net"
	"os"
	"strconv"

	"buf.build/gen/go/chain4travel/camino-messenger-protocol/grpc/go/cmp/services/activity/v1alpha1/activityv1alpha1grpc"
	"buf.build/gen/go/chain4travel/camino-messenger-protocol/grpc/go/cmp/services/network/v1alpha1/networkv1alpha1grpc"
	"buf.build/gen/go/chain4travel/camino-messenger-protocol/grpc/go/cmp/services/partner/v1alpha1/partnerv1alpha1grpc"
	"buf.build/gen/go/chain4travel/camino-messenger-protocol/grpc/go/cmp/services/ping/v1alpha1/pingv1alpha1grpc"
	"buf.build/gen/go/chain4travel/camino-messenger-protocol/grpc/go/cmp/services/transport/v1alpha1/transportv1alpha1grpc"
	activityv1alpha1 "buf.build/gen/go/chain4travel/camino-messenger-protocol/protocolbuffers/go/cmp/services/activity/v1alpha1"
	networkv1alpha1 "buf.build/gen/go/chain4travel/camino-messenger-protocol/protocolbuffers/go/cmp/services/network/v1alpha1"
	partnerv1alpha1 "buf.build/gen/go/chain4travel/camino-messenger-protocol/protocolbuffers/go/cmp/services/partner/v1alpha1"
	pingv1alpha1 "buf.build/gen/go/chain4travel/camino-messenger-protocol/protocolbuffers/go/cmp/services/ping/v1alpha1"
	transportv1alpha1 "buf.build/gen/go/chain4travel/camino-messenger-protocol/protocolbuffers/go/cmp/services/transport/v1alpha1"
	typesv1alpha1 "buf.build/gen/go/chain4travel/camino-messenger-protocol/protocolbuffers/go/cmp/types/v1alpha1"

	"buf.build/gen/go/chain4travel/camino-messenger-protocol/grpc/go/cmp/services/accommodation/v1alpha1/accommodationv1alpha1grpc"
	accommodationv1alpha1 "buf.build/gen/go/chain4travel/camino-messenger-protocol/protocolbuffers/go/cmp/services/accommodation/v1alpha1"
	"github.com/chain4travel/camino-messenger-bot/internal/metadata"
	"google.golang.org/grpc"
)

type partnerPlugin struct {
	activityv1alpha1grpc.ActivitySearchServiceServer
	accommodationv1alpha1grpc.AccommodationSearchServiceServer
	networkv1alpha1grpc.GetNetworkFeeServiceServer
	partnerv1alpha1grpc.GetPartnerConfigurationServiceServer
	pingv1alpha1grpc.PingServiceServer
	transportv1alpha1grpc.TransportSearchServiceServer
}

func (p *partnerPlugin) ActivitySearch(ctx context.Context, request *activityv1alpha1.ActivitySearchRequest) (*activityv1alpha1.ActivitySearchResponse, error) {
	md := metadata.Metadata{}
	err := md.ExtractMetadata(ctx)
	if err != nil {
		log.Print("error extracting metadata")
	}
	md.Stamp(fmt.Sprintf("%s-%s", "ext-system", "response"))
	log.Printf("Responding to request: %s", md.RequestID)

	response := activityv1alpha1.ActivitySearchResponse{
		Header:   nil,
		Metadata: &typesv1alpha1.SearchResponseMetadata{SearchId: &typesv1alpha1.UUID{Value: md.RequestID}},
	}
	grpc.SendHeader(ctx, md.ToGrpcMD())
	return &response, nil
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
		Header:   nil,
		Metadata: &typesv1alpha1.SearchResponseMetadata{SearchId: &typesv1alpha1.UUID{Value: md.RequestID}},
	}
	grpc.SendHeader(ctx, md.ToGrpcMD())
	return &response, nil
}

func (p *partnerPlugin) GetNetworkFee(ctx context.Context, request *networkv1alpha1.GetNetworkFeeRequest) (*networkv1alpha1.GetNetworkFeeResponse, error) {
	md := metadata.Metadata{}
	err := md.ExtractMetadata(ctx)
	if err != nil {
		log.Print("error extracting metadata")
	}
	md.Stamp(fmt.Sprintf("%s-%s", "ext-system", "response"))
	log.Printf("Responding to request: %s", md.RequestID)

	response := networkv1alpha1.GetNetworkFeeResponse{
		NetworkFee: &networkv1alpha1.NetworkFee{
			Amount: 0,
			Asset:  nil,
		},
		CurrentBlockHeight: request.BlockHeight,
	}
	grpc.SendHeader(ctx, md.ToGrpcMD())
	return &response, nil
}
func (p *partnerPlugin) GetPartnerConfiguration(ctx context.Context, request *partnerv1alpha1.GetPartnerConfigurationRequest) (*partnerv1alpha1.GetPartnerConfigurationResponse, error) {
	md := metadata.Metadata{}
	err := md.ExtractMetadata(ctx)
	if err != nil {
		log.Print("error extracting metadata")
	}
	md.Stamp(fmt.Sprintf("%s-%s", "ext-system", "response"))
	log.Printf("Responding to request: %s", md.RequestID)

	response := partnerv1alpha1.GetPartnerConfigurationResponse{
		PartnerConfiguration: &partnerv1alpha1.PartnerConfiguration{},
		CurrentBlockHeight:   request.GetBlockHeight(),
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
func (p *partnerPlugin) TransportSearch(ctx context.Context, request *transportv1alpha1.TransportSearchRequest) (*transportv1alpha1.TransportSearchResponse, error) {
	md := metadata.Metadata{}
	err := md.ExtractMetadata(ctx)
	if err != nil {
		log.Print("error extracting metadata")
	}
	md.Stamp(fmt.Sprintf("%s-%s", "ext-system", "response"))
	log.Printf("Responding to request: %s", md.RequestID)

	response := transportv1alpha1.TransportSearchResponse{
		Header:   nil,
		Metadata: &typesv1alpha1.SearchResponseMetadata{SearchId: &typesv1alpha1.UUID{Value: md.RequestID}},
	}
	grpc.SendHeader(ctx, md.ToGrpcMD())
	return &response, nil
}

func main() {
	grpcServer := grpc.NewServer()
	activityv1alpha1grpc.RegisterActivitySearchServiceServer(grpcServer, &partnerPlugin{})
	accommodationv1alpha1grpc.RegisterAccommodationSearchServiceServer(grpcServer, &partnerPlugin{})
	networkv1alpha1grpc.RegisterGetNetworkFeeServiceServer(grpcServer, &partnerPlugin{})
	partnerv1alpha1grpc.RegisterGetPartnerConfigurationServiceServer(grpcServer, &partnerPlugin{})
	pingv1alpha1grpc.RegisterPingServiceServer(grpcServer, &partnerPlugin{})
	transportv1alpha1grpc.RegisterTransportSearchServiceServer(grpcServer, &partnerPlugin{})
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
