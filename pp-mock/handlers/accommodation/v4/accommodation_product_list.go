// Copyright (C) 2022-2025, Chain4Travel AG. All rights reserved.
// See the file LICENSE for licensing terms.

package v4

import (
	"context"

	"buf.build/gen/go/chain4travel/camino-messenger-protocol/grpc/go/cmp/services/accommodation/v4/accommodationv4grpc"
	accommodationv4 "buf.build/gen/go/chain4travel/camino-messenger-protocol/protocolbuffers/go/cmp/services/accommodation/v4"
	typesv4 "buf.build/gen/go/chain4travel/camino-messenger-protocol/protocolbuffers/go/cmp/types/v4"
	mockdata "github.com/chain4travel/camino-messenger-bot/v11/pp-mock/services/data"
)

var _ accommodationv4grpc.AccommodationProductListServiceServer = (*accommodationProductListV3Server)(nil)

type accommodationProductListV3Server struct{}

func NewAccommodationProductListServer() accommodationv4grpc.AccommodationProductListServiceServer {
	return &accommodationProductListV3Server{}
}

func (s *accommodationProductListV3Server) AccommodationProductList(_ context.Context, req *accommodationv4.AccommodationProductListRequest) (*accommodationv4.AccommodationProductListResponse, error) {
	filteredProperties := filterPropertiesByLastModified(mockdata.PropertiesV4, req.GetModifiedAfter().AsTime())

	response := &accommodationv4.AccommodationProductListResponse{
		Header: &typesv4.ResponseHeader{
			BaseHeader: &typesv4.Header{Version: &typesv4.Version{}},
			Status:     typesv4.StatusType_STATUS_TYPE_SUCCESS,
		},
		Properties: filteredProperties,
	}

	if len(filteredProperties) == 0 {
		response.Header.Alerts = []*typesv4.Alert{{
			Message: "No properties found that match request",
			Type:    typesv4.AlertType_ALERT_TYPE_INFO,
		}}
	}

	return response, nil
}
