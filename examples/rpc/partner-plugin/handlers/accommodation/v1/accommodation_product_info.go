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
	var suppliersFiltered []*accommodationv1.PropertyExtendedInfo

	// check if there are supplier codes in the request
	if req.SupplierCodes != nil {
		log.Printf("Supplier codes requested: %v", req.SupplierCodes)
		// filter properties by supplier codes
		for _, property := range properties {
			for _, supplierCode := range req.SupplierCodes {
				if property.Property.SupplierCode.SupplierCode == supplierCode.SupplierCode {
					suppliersFiltered = append(suppliersFiltered, &property)
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

	// Initialize filteredProperties slice for language filtering
	var filteredProperties []*accommodationv1.PropertyExtendedInfo

	// check if there is a language in the request
	if req.Languages != nil {
		log.Printf("Languages requested: %v", req.Languages)

		// loop only on suppliersFiltered if supplier codes were requested
		if req.SupplierCodes != nil {
			properties = make([]accommodationv1.PropertyExtendedInfo, len(suppliersFiltered))
			for i, p := range suppliersFiltered {
				properties[i] = *p
			}
		}

		// filter properties by language
		for _, property := range properties {
			// Check if property has any description matching requested languages
			for _, reqLang := range req.Languages {
				for _, desc := range property.LocalizedDescriptions {
					if desc.Language == reqLang {
						// check if already in filteredProperties
						if !containsProperty(filteredProperties, property) {
							filteredProperties = append(filteredProperties, &property)
						}
					} else {
						// check if the propery is already added to the list and remove it (should filter by language)
						if containsProperty(filteredProperties, property) {
							filteredProperties = removeProperty(filteredProperties, property)
						}
					}
				}
			}
		}
	} else {
		// If no language is requested, use all properties
		filteredProperties = suppliersFiltered
	}

	response := &accommodationv1.AccommodationProductInfoResponse{
		Header:     nil,
		Properties: filteredProperties,
	}

	log.Printf("CMAccount %s received request from CMAccount %s", md.Recipient, md.Sender)

	grpc.SendHeader(ctx, md.ToGrpcMD())

	return response, nil
}

// containsProperty checks if a property already exists in the slice
func containsProperty(properties []*accommodationv1.PropertyExtendedInfo, property accommodationv1.PropertyExtendedInfo) bool {
	for _, p := range properties {
		if p.Property.SupplierCode.SupplierCode == property.Property.SupplierCode.SupplierCode {
			return true
		}
	}
	return false
}

// removeProperty removes a property from the slice and returns the updated slice
func removeProperty(properties []*accommodationv1.PropertyExtendedInfo, property accommodationv1.PropertyExtendedInfo) []*accommodationv1.PropertyExtendedInfo {
	result := make([]*accommodationv1.PropertyExtendedInfo, 0)
	for _, p := range properties {
		if p.Property.SupplierCode.SupplierCode != property.Property.SupplierCode.SupplierCode {
			result = append(result, p)
		}
	}
	return result
}
