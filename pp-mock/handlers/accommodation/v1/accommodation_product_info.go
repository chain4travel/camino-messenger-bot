// Copyright (C) 2022-2025, Chain4Travel AG. All rights reserved.
// See the file LICENSE for licensing terms.

package v1

import (
	"context"

	"buf.build/gen/go/chain4travel/camino-messenger-protocol/grpc/go/cmp/services/accommodation/v1/accommodationv1grpc"
	accommodationv1 "buf.build/gen/go/chain4travel/camino-messenger-protocol/protocolbuffers/go/cmp/services/accommodation/v1"
	typesv1 "buf.build/gen/go/chain4travel/camino-messenger-protocol/protocolbuffers/go/cmp/types/v1"
	mockdata "github.com/chain4travel/camino-messenger-bot/v11/pp-mock/services/data"
)

var _ accommodationv1grpc.AccommodationProductInfoServiceServer = (*accommodationProductInfoV1Server)(nil)

type accommodationProductInfoV1Server struct{}

func NewAccommodationProductInfoServer() accommodationv1grpc.AccommodationProductInfoServiceServer {
	return &accommodationProductInfoV1Server{}
}

func (s *accommodationProductInfoV1Server) AccommodationProductInfo(_ context.Context, req *accommodationv1.AccommodationProductInfoRequest) (*accommodationv1.AccommodationProductInfoResponse, error) {
	filteredProperties := filterExtendedPropertiesBySupplierCodes(mockdata.PropertiesV1, req.SupplierCodes)
	filteredProperties = filterExtendedPropertiesByLanguage(filteredProperties, req.Languages)

	response := &accommodationv1.AccommodationProductInfoResponse{
		Header: &typesv1.ResponseHeader{
			Status: typesv1.StatusType_STATUS_TYPE_SUCCESS,
		},
		Properties: filteredProperties,
	}

	if len(filteredProperties) == 0 {
		response.Header.Alerts = []*typesv1.Alert{{
			Message: "No properties found that match request",
			Type:    typesv1.AlertType_ALERT_TYPE_INFO,
		}}
	}

	return response, nil
}
