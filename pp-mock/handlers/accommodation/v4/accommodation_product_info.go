// Copyright (C) 2022-2025, Chain4Travel AG. All rights reserved.
// See the file LICENSE for licensing terms.

package v4

import (
	"context"

	"buf.build/gen/go/chain4travel/camino-messenger-protocol/grpc/go/cmp/services/accommodation/v4/accommodationv4grpc"
	accommodationv4 "buf.build/gen/go/chain4travel/camino-messenger-protocol/protocolbuffers/go/cmp/services/accommodation/v4"
	"github.com/chain4travel/camino-messenger-bot/v11/pp-mock/common"
	mockdata "github.com/chain4travel/camino-messenger-bot/v11/pp-mock/services/data"
)

var _ accommodationv4grpc.AccommodationProductInfoServiceServer = (*accommodationProductInfoV4Server)(nil)

type accommodationProductInfoV4Server struct{}

func NewAccommodationProductInfoServer() accommodationv4grpc.AccommodationProductInfoServiceServer {
	return &accommodationProductInfoV4Server{}
}

func (s *accommodationProductInfoV4Server) AccommodationProductInfo(_ context.Context, req *accommodationv4.AccommodationProductInfoRequest) (*accommodationv4.AccommodationProductInfoResponse, error) {
	filteredProperties := filterExtendedPropertiesBySupplierCodes(mockdata.PropertiesV4, req.SupplierCodes)
	filteredProperties = filterExtendedPropertiesByLanguage(filteredProperties, req.Languages)

	response := &accommodationv4.AccommodationProductInfoResponse{
		Header:     common.SuccessHeaderV4(),
		Properties: filteredProperties,
	}

	if len(filteredProperties) == 0 {
		common.AddHeaderInfoV4(response.Header, "No properties found that match request")
	}

	return response, nil
}
