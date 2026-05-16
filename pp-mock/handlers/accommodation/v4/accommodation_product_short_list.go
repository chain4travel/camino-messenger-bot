// Copyright (C) 2022-2026, Chain4Travel AG. All rights reserved.
// See the file LICENSE for licensing terms.

package v4

import (
	"context"

	"buf.build/gen/go/chain4travel/camino-messenger-protocol/grpc/go/cmp/services/accommodation/v4/accommodationv4grpc"
	accommodationv4 "buf.build/gen/go/chain4travel/camino-messenger-protocol/protocolbuffers/go/cmp/services/accommodation/v4"
	typesv4 "buf.build/gen/go/chain4travel/camino-messenger-protocol/protocolbuffers/go/cmp/types/v4"
	"github.com/chain4travel/camino-messenger-bot/v13/pp-mock/common"
	mockdata "github.com/chain4travel/camino-messenger-bot/v13/pp-mock/services/data"
)

var _ accommodationv4grpc.AccommodationProductShortListServiceServer = (*accommodationProductShortListV4Server)(nil)

type accommodationProductShortListV4Server struct{}

func NewAccommodationProductShortListServer() accommodationv4grpc.AccommodationProductShortListServiceServer {
	return &accommodationProductShortListV4Server{}
}

func (s *accommodationProductShortListV4Server) AccommodationProductShortList(_ context.Context, req *accommodationv4.AccommodationProductShortListRequest) (*accommodationv4.AccommodationProductShortListResponse, error) {
	filteredProperties := filterShortItemsByModifierAfter(mockdata.PropertiesV4, req.ModifiedAfter.AsTime())

	response := &accommodationv4.AccommodationProductShortListResponse{
		Response: &accommodationv4.AccommodationProductShortListResponse_SuccessResponse{
			SuccessResponse: &accommodationv4.AccommodationProductShortListSuccessResponse{
				Header:                 common.SuccessHeaderV4(),
				PropertyShortListItems: filteredProperties,
			},
		},
	}

	if len(filteredProperties) == 0 {
		common.AddHeaderAlertV4(response.GetSuccessResponse().Header, typesv4.AlertCode_ALERT_CODE_NO_CONTENT, "No properties found that match request")
	}

	return response, nil
}
