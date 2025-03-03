// Copyright (C) 2022-2025, Chain4Travel AG. All rights reserved.
// See the file LICENSE for licensing terms.

package handlers

import (
	"context"
	"fmt"
	"log"

	"buf.build/gen/go/chain4travel/camino-messenger-protocol/grpc/go/cmp/services/accommodation/v3/accommodationv3grpc"
	accommodationv3 "buf.build/gen/go/chain4travel/camino-messenger-protocol/protocolbuffers/go/cmp/services/accommodation/v3"
	typesv1 "buf.build/gen/go/chain4travel/camino-messenger-protocol/protocolbuffers/go/cmp/types/v1"
	"github.com/chain4travel/camino-messenger-bot/internal/metadata"
	mockdata "github.com/chain4travel/camino-messenger-bot/pp-mock/services/data"
	"google.golang.org/grpc"
	"google.golang.org/protobuf/proto"
)

var _ accommodationv3grpc.AccommodationProductInfoServiceServer = (*AccommodationProductInfoV3Server)(nil)

type AccommodationProductInfoV3Server struct{}

func (*AccommodationProductInfoV3Server) AccommodationProductInfo(ctx context.Context, req *accommodationv3.AccommodationProductInfoRequest) (*accommodationv3.AccommodationProductInfoResponse, error) {
	md := metadata.Metadata{}

	if err := md.ExtractMetadata(ctx); err != nil {
		log.Print("error extracting metadata")
	}

	md.Stamp(fmt.Sprintf("%s-%s", "ext-system", "response"))

	log.Printf("Responding to request (Accommodation Product Info): %s", md.RequestID)

	// Initialize suppliersFiltered with the correct type
	var suppliersFiltered []*accommodationv3.PropertyExtendedInfo

	// check if there are supplier codes in the request
	if req.SupplierCodes != nil {
		log.Printf("Supplier codes requested: %v", req.SupplierCodes)
		suppliersFiltered = []*accommodationv3.PropertyExtendedInfo{}
		// filter properties by supplier codes
		for _, property := range mockdata.PropertiesV3 {
			for _, supplierCode := range req.SupplierCodes {
				if property.Property.SupplierCode.SupplierCode == supplierCode.SupplierCode {
					suppliersFiltered = append(
						suppliersFiltered,
						proto.Clone(property).(*accommodationv3.PropertyExtendedInfo),
					)
				}
			}
		}
	} else {
		suppliersFiltered = make([]*accommodationv3.PropertyExtendedInfo, len(mockdata.PropertiesV3))
		copy(suppliersFiltered, mockdata.PropertiesV3)
	}

	filteredProperties := []*accommodationv3.PropertyExtendedInfo{}

	if len(req.Languages) > 0 {
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

	if len(filteredProperties) == 0 {
		return &accommodationv3.AccommodationProductInfoResponse{
			Header: &typesv1.ResponseHeader{
				Status: typesv1.StatusType_STATUS_TYPE_SUCCESS,
				Alerts: []*typesv1.Alert{{
					Message: fmt.Sprintf("No properties found for supplier codes: %v", req.SupplierCodes),
					Type:    typesv1.AlertType_ALERT_TYPE_INFO,
				}},
			},
		}, nil
	}

	response := &accommodationv3.AccommodationProductInfoResponse{
		Header: &typesv1.ResponseHeader{
			Status: typesv1.StatusType_STATUS_TYPE_SUCCESS,
		},
		Properties: filteredProperties,
	}

	log.Printf("CMAccount %s received request from CMAccount %s", md.Recipient, md.Sender)

	if err := grpc.SetHeader(ctx, md.ToGrpcMD()); err != nil {
		log.Printf("Failed to set header: %v", err)
	}

	return response, nil
}

// containsProperty checks if a property already exists in the slice
func containsProperty(properties []*accommodationv3.PropertyExtendedInfo, property *accommodationv3.PropertyExtendedInfo) bool {
	for _, p := range properties {
		if p.Property.SupplierCode.SupplierCode == property.Property.SupplierCode.SupplierCode {
			return true
		}
	}
	return false
}
