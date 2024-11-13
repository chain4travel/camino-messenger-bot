package handlers

import (
	"context"
	"fmt"
	"log"

	"buf.build/gen/go/chain4travel/camino-messenger-protocol/grpc/go/cmp/services/accommodation/v2/accommodationv2grpc"
	accommodationv2 "buf.build/gen/go/chain4travel/camino-messenger-protocol/protocolbuffers/go/cmp/services/accommodation/v2"
	typesv1 "buf.build/gen/go/chain4travel/camino-messenger-protocol/protocolbuffers/go/cmp/types/v1"
	mock_data "github.com/chain4travel/camino-messenger-bot/examples/rpc/partner-plugin/services/data/v2"
	"github.com/chain4travel/camino-messenger-bot/internal/metadata"
	"google.golang.org/grpc"
)

var _ accommodationv2grpc.AccommodationProductInfoServiceServer = (*AccommodationProductInfoV2Server)(nil)

type AccommodationProductInfoV2Server struct{}

func (*AccommodationProductInfoV2Server) AccommodationProductInfo(ctx context.Context, req *accommodationv2.AccommodationProductInfoRequest) (*accommodationv2.AccommodationProductInfoResponse, error) {
	md := metadata.Metadata{}

	if err := md.ExtractMetadata(ctx); err != nil {
		log.Print("error extracting metadata")
	}

	md.Stamp(fmt.Sprintf("%s-%s", "ext-system", "response"))

	log.Printf("Responding to request (Accommodation Product Info): %s", md.RequestID)

	// Load properties data
	properties := mock_data.LoadPropertiesMockData()

	// Initialize suppliersFiltered with the correct type
	suppliersFiltered := []*accommodationv2.PropertyExtendedInfo{}

	// check if there are supplier codes in the request
	if req.SupplierCodes != nil {
		log.Printf("Supplier codes requested: %v", req.SupplierCodes)
		// filter properties by supplier codes
		for i := range properties {
			property := &properties[i]
			for _, supplierCode := range req.SupplierCodes {
				if property.Property.SupplierCode.SupplierCode == supplierCode.SupplierCode {
					suppliersFiltered = append(suppliersFiltered, property)
				}
			}
		}
	} else {
		// Convert []accommodationv2.PropertyExtendedInfo to []*accommodationv2.PropertyExtendedInfo
		suppliersFiltered = make([]*accommodationv2.PropertyExtendedInfo, len(properties))
		for i := range properties {
			suppliersFiltered[i] = &properties[i]
		}
	}

	filteredProperties := []*accommodationv2.PropertyExtendedInfo{}

	if req.Languages != nil {
		log.Printf("Languages requested: %v", req.Languages)

		for i := range suppliersFiltered {
			property := &suppliersFiltered[i]
			filteredDescriptions := []*typesv1.LocalizedDescriptionSet{}
			filteredRoomDescriptions := []*typesv1.LocalizedDescriptionSet{}

			for _, descSet := range (*property).LocalizedDescriptions {
				for _, reqLang := range req.Languages {
					if descSet.Language == reqLang {
						filteredDescriptions = append(filteredDescriptions, descSet)
						break
					}
				}
			}
			for _, roomDescSet := range (*property).LocalizedRoomDescriptions {
				for _, reqLang := range req.Languages {
					if roomDescSet.Language == reqLang {
						filteredRoomDescriptions = append(filteredRoomDescriptions, roomDescSet)
						break
					}
				}
			}

			if (len(filteredDescriptions) > 0 || len(filteredRoomDescriptions) > 0) && !containsProperty(filteredProperties, *property) {
				(*property).LocalizedDescriptions = filteredDescriptions
				(*property).LocalizedRoomDescriptions = filteredRoomDescriptions
				filteredProperties = append(filteredProperties, *property)
			}
		}
	} else {
		filteredProperties = suppliersFiltered
	}
	response := &accommodationv2.AccommodationProductInfoResponse{
		Header:     &typesv1.ResponseHeader{},
		Properties: filteredProperties,
	}

	log.Printf("CMAccount %s received request from CMAccount %s", md.Recipient, md.Sender)

	// reload properties data
	if err := mock_data.ReloadPropertiesMockData(); err != nil {
		log.Printf("Error reloading properties data: %v", err)
	}

	grpc.SendHeader(ctx, md.ToGrpcMD())

	return response, nil
}

// containsProperty checks if a property already exists in the slice
func containsProperty(properties []*accommodationv2.PropertyExtendedInfo, property *accommodationv2.PropertyExtendedInfo) bool {
	for _, p := range properties {
		if p.Property.SupplierCode.SupplierCode == property.Property.SupplierCode.SupplierCode {
			return true
		}
	}
	return false
}
