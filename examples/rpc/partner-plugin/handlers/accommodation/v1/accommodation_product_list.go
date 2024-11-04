package handlers

import (
	"context"
	"fmt"
	"log"

	"buf.build/gen/go/chain4travel/camino-messenger-protocol/grpc/go/cmp/services/accommodation/v1/accommodationv1grpc"
	accommodationv1 "buf.build/gen/go/chain4travel/camino-messenger-protocol/protocolbuffers/go/cmp/services/accommodation/v1"
	helpers "github.com/chain4travel/camino-messenger-bot/examples/rpc/partner-plugin/helpers"
	"github.com/chain4travel/camino-messenger-bot/internal/metadata"
	"google.golang.org/grpc"
)

var _ accommodationv1grpc.AccommodationProductListServiceServer = (*AccommodationProductListV1Server)(nil)

type AccommodationProductListV1Server struct{}

func (*AccommodationProductListV1Server) AccommodationProductList(ctx context.Context, req *accommodationv1.AccommodationProductListRequest) (*accommodationv1.AccommodationProductListResponse, error) {
	md := metadata.Metadata{}

	if err := md.ExtractMetadata(ctx); err != nil {
		log.Print("error extracting metadata")
	}

	md.Stamp(fmt.Sprintf("%s-%s", "ext-system", "response"))

	log.Printf("Responding to request (Accommodation Product List): %s", md.RequestID)

	// Load properties data
	properties := helpers.LoadPropertiesMockData()

	// filter only property objects
	filteredProperties := []*accommodationv1.Property{}
	for _, property := range properties {
		filteredProperties = append(filteredProperties, property.Property)
	}

	response := &accommodationv1.AccommodationProductListResponse{
		Header:     nil,
		Properties: filteredProperties,
	}

	log.Printf("CMAccount %s received request from CMAccount %s", md.Recipient, md.Sender)

	grpc.SendHeader(ctx, md.ToGrpcMD())

	return response, nil
}
