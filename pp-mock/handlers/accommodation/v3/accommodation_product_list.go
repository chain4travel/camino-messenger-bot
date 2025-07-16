// Copyright (C) 2022-2025, Chain4Travel AG. All rights reserved.
// See the file LICENSE for licensing terms.

package v3

import (
	"context"

	"buf.build/gen/go/chain4travel/camino-messenger-protocol/grpc/go/cmp/services/accommodation/v3/accommodationv3grpc"
	accommodationv3 "buf.build/gen/go/chain4travel/camino-messenger-protocol/protocolbuffers/go/cmp/services/accommodation/v3"
	typesv1 "buf.build/gen/go/chain4travel/camino-messenger-protocol/protocolbuffers/go/cmp/types/v1"
	mockdata "github.com/chain4travel/camino-messenger-bot/v11/pp-mock/services/data"
)

var _ accommodationv3grpc.AccommodationProductListServiceServer = (*accommodationProductListV3Server)(nil)

type accommodationProductListV3Server struct{}

func NewAccommodationProductListV3Server() accommodationv3grpc.AccommodationProductListServiceServer {
	return &accommodationProductListV3Server{}
}

func (s *accommodationProductListV3Server) AccommodationProductList(_ context.Context, req *accommodationv3.AccommodationProductListRequest) (*accommodationv3.AccommodationProductListResponse, error) {
	filteredProperties := filterPropertiesByLastModified(mockdata.PropertiesV3, req.GetModifiedAfter().AsTime())

	response := &accommodationv3.AccommodationProductListResponse{
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
