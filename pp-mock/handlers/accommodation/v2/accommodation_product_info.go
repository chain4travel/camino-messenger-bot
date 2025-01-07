// Copyright (C) 2022-2025, Chain4Travel AG. All rights reserved.
// See the file LICENSE for licensing terms.

package handlers

import (
	"context"
	"fmt"
	"log"

	"buf.build/gen/go/chain4travel/camino-messenger-protocol/grpc/go/cmp/services/accommodation/v2/accommodationv2grpc"
	accommodationv2 "buf.build/gen/go/chain4travel/camino-messenger-protocol/protocolbuffers/go/cmp/services/accommodation/v2"
	typesv1 "buf.build/gen/go/chain4travel/camino-messenger-protocol/protocolbuffers/go/cmp/types/v1"
	"github.com/chain4travel/camino-messenger-bot/internal/metadata"
	mockdata "github.com/chain4travel/camino-messenger-bot/pp-mock/services/data"
	"google.golang.org/grpc"
	"google.golang.org/protobuf/proto"
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

	// Initialize suppliersFiltered with the correct type
	suppliersFiltered := []*accommodationv2.PropertyExtendedInfo{}

	// check if there are supplier codes in the request
	if req.SupplierCodes != nil {
		log.Printf("Supplier codes requested: %v", req.SupplierCodes)
		// filter properties by supplier codes
		for _, property := range mockdata.PropertiesV2 {
			for _, supplierCode := range req.SupplierCodes {
				if property.Property.SupplierCode.SupplierCode == supplierCode.SupplierCode {
					suppliersFiltered = append(
						suppliersFiltered,
						proto.Clone(property).(*accommodationv2.PropertyExtendedInfo),
					)
				}
			}
		}
	} else {
		for i := range mockdata.PropertiesV2 {
			suppliersFiltered[i] = proto.Clone(mockdata.PropertiesV1[i]).(*accommodationv2.PropertyExtendedInfo)
		}
	}

	filteredProperties := []*accommodationv2.PropertyExtendedInfo{}

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
		return &accommodationv2.AccommodationProductInfoResponse{
			Header: &typesv1.ResponseHeader{
				Status: typesv1.StatusType_STATUS_TYPE_SUCCESS,
				Alerts: []*typesv1.Alert{
					{
						Message: fmt.Sprintf("No properties found for supplier codes: %v", req.SupplierCodes),
						Type:    typesv1.AlertType_ALERT_TYPE_INFO,
					},
				},
			},
		}, nil
	}

	response := &accommodationv2.AccommodationProductInfoResponse{
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
func containsProperty(properties []*accommodationv2.PropertyExtendedInfo, property *accommodationv2.PropertyExtendedInfo) bool {
	for _, p := range properties {
		if p.Property.SupplierCode.SupplierCode == property.Property.SupplierCode.SupplierCode {
			return true
		}
	}
	return false
}
