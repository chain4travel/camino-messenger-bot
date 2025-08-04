// Copyright (C) 2022-2025, Chain4Travel AG. All rights reserved.
// See the file LICENSE for licensing terms.

package v2

import (
	"context"

	"buf.build/gen/go/chain4travel/camino-messenger-protocol/grpc/go/cmp/services/accommodation/v2/accommodationv2grpc"
	accommodationv2 "buf.build/gen/go/chain4travel/camino-messenger-protocol/protocolbuffers/go/cmp/services/accommodation/v2"
	typesv1 "buf.build/gen/go/chain4travel/camino-messenger-protocol/protocolbuffers/go/cmp/types/v1"
	mockdata "github.com/chain4travel/camino-messenger-bot/v11/pp-mock/services/data"
)

var _ accommodationv2grpc.AccommodationProductListServiceServer = (*accommodationProductListV2Server)(nil)

type accommodationProductListV2Server struct{}

func NewAccommodationProductListServer() accommodationv2grpc.AccommodationProductListServiceServer {
	return &accommodationProductListV2Server{}
}

func (s *accommodationProductListV2Server) AccommodationProductList(_ context.Context, req *accommodationv2.AccommodationProductListRequest) (*accommodationv2.AccommodationProductListResponse, error) {
	filteredProperties := filterPropertiesByLastModified(mockdata.PropertiesV2, req.GetModifiedAfter().AsTime())

	response := &accommodationv2.AccommodationProductListResponse{
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
