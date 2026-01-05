// Copyright (C) 2022-2026, Chain4Travel AG. All rights reserved.
// See the file LICENSE for licensing terms.

package v4

import (
	"context"

	"buf.build/gen/go/chain4travel/camino-messenger-protocol/grpc/go/cmp/services/accommodation/v4/accommodationv4grpc"
	accommodationv4 "buf.build/gen/go/chain4travel/camino-messenger-protocol/protocolbuffers/go/cmp/services/accommodation/v4"
	typesv4 "buf.build/gen/go/chain4travel/camino-messenger-protocol/protocolbuffers/go/cmp/types/v4"
	"github.com/chain4travel/camino-messenger-bot/v12/pp-mock/common"
	mockdata "github.com/chain4travel/camino-messenger-bot/v12/pp-mock/services/data"
)

var _ accommodationv4grpc.AccommodationProductListServiceServer = (*accommodationProductListV4Server)(nil)

type accommodationProductListV4Server struct{}

func NewAccommodationProductListServer() accommodationv4grpc.AccommodationProductListServiceServer {
	return &accommodationProductListV4Server{}
}

func (s *accommodationProductListV4Server) AccommodationProductList(_ context.Context, req *accommodationv4.AccommodationProductListRequest) (*accommodationv4.AccommodationProductListResponse, error) {
	filteredProperties := filterPropertiesBySupplierCodes(mockdata.PropertiesV4, req.SupplierCodes)

	response := &accommodationv4.AccommodationProductListResponse{
		Response: &accommodationv4.AccommodationProductListResponse_SuccessResponse{
			SuccessResponse: &accommodationv4.AccommodationProductListSuccessResponse{
				Header:     common.SuccessHeaderV4(),
				Properties: filteredProperties,
			},
		},
	}

	if len(filteredProperties) == 0 {
		common.AddHeaderAlertV4(response.GetSuccessResponse().Header, typesv4.AlertCode_ALERT_CODE_NO_CONTENT, "No properties found that match request")
	}

	return response, nil
}
