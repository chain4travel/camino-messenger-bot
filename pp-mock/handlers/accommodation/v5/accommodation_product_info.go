// Copyright (C) 2022-2026, Chain4Travel AG. All rights reserved.
// See the file LICENSE for licensing terms.

package v5

import (
	"context"

	"buf.build/gen/go/chain4travel/camino-messenger-protocol/grpc/go/cmp/services/accommodation/v5/accommodationv5grpc"
	accommodationv5 "buf.build/gen/go/chain4travel/camino-messenger-protocol/protocolbuffers/go/cmp/services/accommodation/v5"
	typesv4 "buf.build/gen/go/chain4travel/camino-messenger-protocol/protocolbuffers/go/cmp/types/v4"
	"github.com/chain4travel/camino-messenger-bot/v13/pp-mock/common"
	mockdata "github.com/chain4travel/camino-messenger-bot/v13/pp-mock/services/data"
)

var _ accommodationv5grpc.AccommodationProductInfoServiceServer = (*accommodationProductInfoV5Server)(nil)

type accommodationProductInfoV5Server struct{}

func NewAccommodationProductInfoServer() accommodationv5grpc.AccommodationProductInfoServiceServer {
	return &accommodationProductInfoV5Server{}
}

func (s *accommodationProductInfoV5Server) AccommodationProductInfo(_ context.Context, req *accommodationv5.AccommodationProductInfoRequest) (*accommodationv5.AccommodationProductInfoResponse, error) {
	filteredProperties := filterExtendedPropertiesBySupplierCodes(mockdata.PropertiesV5, req.SupplierCodes)
	filteredProperties = filterExtendedPropertiesByLanguage(filteredProperties, req.Languages)

	response := &accommodationv5.AccommodationProductInfoResponse{
		Response: &accommodationv5.AccommodationProductInfoResponse_SuccessResponse{
			SuccessResponse: &accommodationv5.AccommodationProductInfoSuccessResponse{
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
