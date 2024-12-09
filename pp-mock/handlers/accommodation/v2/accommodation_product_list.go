package handlers

import (
	"context"
	"encoding/json"
	"fmt"
	"log"

	"buf.build/gen/go/chain4travel/camino-messenger-protocol/grpc/go/cmp/services/accommodation/v2/accommodationv2grpc"
	accommodationv2 "buf.build/gen/go/chain4travel/camino-messenger-protocol/protocolbuffers/go/cmp/services/accommodation/v2"
	typesv1 "buf.build/gen/go/chain4travel/camino-messenger-protocol/protocolbuffers/go/cmp/types/v1"
	"github.com/chain4travel/camino-messenger-bot/internal/metadata"
	mock_data "github.com/chain4travel/camino-messenger-bot/pp-mock/services/data"
	"google.golang.org/grpc"
)

var _ accommodationv2grpc.AccommodationProductListServiceServer = (*AccommodationProductListV2Server)(nil)

type AccommodationProductListV2Server struct{}

func (*AccommodationProductListV2Server) AccommodationProductList(ctx context.Context, req *accommodationv2.AccommodationProductListRequest) (*accommodationv2.AccommodationProductListResponse, error) {
	md := metadata.Metadata{}

	if err := md.ExtractMetadata(ctx); err != nil {
		log.Print("error extracting metadata")
	}

	md.Stamp(fmt.Sprintf("%s-%s", "ext-system", "response"))

	log.Printf("Responding to request (Accommodation Product List): %s", md.RequestID)

	// Load properties data
	var properties []accommodationv2.PropertyExtendedInfo
	jsonProperties := mock_data.PropertiesJSON

	// Unmarshal properties
	err := json.Unmarshal([]byte(jsonProperties), &properties)
	if err != nil {
		log.Printf("Error unmarshalling properties: %v", err)
	}

	// filter only property objects
	filteredProperties := []*accommodationv2.Property{}
	for i := range properties {
		property := &properties[i]
		filteredProperties = append(filteredProperties, property.Property)
	}

	response := &accommodationv2.AccommodationProductListResponse{
		Header: &typesv1.ResponseHeader{
			Status: typesv1.StatusType_STATUS_TYPE_SUCCESS,
		},
		Properties: filteredProperties,
	}

	log.Printf("CMAccount %s received request from CMAccount %s", md.Recipient, md.Sender)

	grpc.SendHeader(ctx, md.ToGrpcMD())

	return response, nil
}
