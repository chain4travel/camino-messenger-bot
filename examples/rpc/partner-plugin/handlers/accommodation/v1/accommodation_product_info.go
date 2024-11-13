package handlers

import (
	"context"
	"fmt"
	"log"

	"buf.build/gen/go/chain4travel/camino-messenger-protocol/grpc/go/cmp/services/accommodation/v1/accommodationv1grpc"
	accommodationv1 "buf.build/gen/go/chain4travel/camino-messenger-protocol/protocolbuffers/go/cmp/services/accommodation/v1"
	typesv1 "buf.build/gen/go/chain4travel/camino-messenger-protocol/protocolbuffers/go/cmp/types/v1"
	helpers "github.com/chain4travel/camino-messenger-bot/examples/rpc/partner-plugin/services/data/v1"
	"github.com/chain4travel/camino-messenger-bot/internal/metadata"
	"google.golang.org/grpc"
)

var _ accommodationv1grpc.AccommodationProductInfoServiceServer = (*AccommodationProductInfoV1Server)(nil)

type AccommodationProductInfoV1Server struct{}

func (*AccommodationProductInfoV1Server) AccommodationProductInfo(ctx context.Context, req *accommodationv1.AccommodationProductInfoRequest) (*accommodationv1.AccommodationProductInfoResponse, error) {
	md := metadata.Metadata{}

	if err := md.ExtractMetadata(ctx); err != nil {
		log.Print("error extracting metadata")
	}

	md.Stamp(fmt.Sprintf("%s-%s", "ext-system", "response"))

	log.Printf("Responding to request (Accommodation Product Info): %s", md.RequestID)

	// Load properties data
	properties := helpers.LoadPropertiesMockData()

	// Initialize suppliersFiltered with the correct type
	suppliersFiltered := []*accommodationv1.PropertyExtendedInfo{}

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
		// Convert []accommodationv1.PropertyExtendedInfo to []*accommodationv1.PropertyExtendedInfo
		suppliersFiltered = make([]*accommodationv1.PropertyExtendedInfo, len(properties))
		for i := range properties {
			suppliersFiltered[i] = &properties[i]
		}
	}

	filteredProperties := []*accommodationv1.PropertyExtendedInfo{}

	if req.Languages != nil {
		log.Printf("Languages requested: %v", req.Languages)

		for _, property := range suppliersFiltered {
			filteredDescriptions := []*typesv1.LocalizedDescriptionSet{}
			filteredRoomDescriptions := []*typesv1.LocalizedDescriptionSet{}

			for _, descSet := range property.LocalizedDescriptions {
				for _, reqLang := range req.Languages {
					if descSet.Language == reqLang {
						filteredDescriptions = append(filteredDescriptions, descSet)
						break
					}
				}
			}
			for _, roomDescSet := range property.LocalizedRoomDescriptions {
				for _, reqLang := range req.Languages {
					if roomDescSet.Language == reqLang {
						filteredRoomDescriptions = append(filteredRoomDescriptions, roomDescSet)
						break
					}
				}
			}

			if (len(filteredDescriptions) > 0 || len(filteredRoomDescriptions) > 0) && !containsProperty(filteredProperties, property) {
				property.LocalizedDescriptions = filteredDescriptions
				property.LocalizedRoomDescriptions = filteredRoomDescriptions
				filteredProperties = append(filteredProperties, property)
			}
		}
	} else {
		filteredProperties = suppliersFiltered
	}

	response := &accommodationv1.AccommodationProductInfoResponse{
		Header: &typesv1.ResponseHeader{
			Status: typesv1.StatusType_STATUS_TYPE_SUCCESS,
		},
		Properties: filteredProperties,
	}

	// reload properties data
	if err := helpers.ReloadPropertiesMockData(); err != nil {
		log.Printf("Error reloading properties data: %v", err)
	}

	log.Printf("CMAccount %s received request from CMAccount %s", md.Recipient, md.Sender)

	grpc.SendHeader(ctx, md.ToGrpcMD())

	return response, nil
}

// containsProperty checks if a property already exists in the slice
func containsProperty(properties []*accommodationv1.PropertyExtendedInfo, property *accommodationv1.PropertyExtendedInfo) bool {
	for _, p := range properties {
		if p.Property.SupplierCode.SupplierCode == property.Property.SupplierCode.SupplierCode {
			return true
		}
	}
	return false
}
