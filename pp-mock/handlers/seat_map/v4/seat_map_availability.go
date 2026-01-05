// Copyright (C) 2022-2026, Chain4Travel AG. All rights reserved.
// See the file LICENSE for licensing terms.

package v4

import (
	"context"

	"buf.build/gen/go/chain4travel/camino-messenger-protocol/grpc/go/cmp/services/seat_map/v4/seat_mapv4grpc"
	seat_mapv4 "buf.build/gen/go/chain4travel/camino-messenger-protocol/protocolbuffers/go/cmp/services/seat_map/v4"
	typesv4 "buf.build/gen/go/chain4travel/camino-messenger-protocol/protocolbuffers/go/cmp/types/v4"
	"github.com/chain4travel/camino-messenger-bot/v12/pp-mock/common"
	"github.com/chain4travel/camino-messenger-bot/v12/pp-mock/handlers/state"
	mockdata "github.com/chain4travel/camino-messenger-bot/v12/pp-mock/services/data"
)

var _ seat_mapv4grpc.SeatMapAvailabilityServiceServer = (*seatMapAvailabilityV4Server)(nil)

type seatMapAvailabilityV4Server struct{}

func NewSeatMapAvailabilityServer() seat_mapv4grpc.SeatMapAvailabilityServiceServer {
	return &seatMapAvailabilityV4Server{}
}

func (s *seatMapAvailabilityV4Server) SeatMapAvailability(_ context.Context, req *seat_mapv4.SeatMapAvailabilityRequest) (*seat_mapv4.SeatMapAvailabilityResponse, error) {
	seatMapID := ""
	switch identifier := req.Identifier.(type) {
	case *seat_mapv4.SeatMapAvailabilityRequest_SearchResultIdentifier:
		storedMintData, found := state.GetStore().GetSearchResult(identifier.SearchResultIdentifier.SearchId.Value)
		if found {
			seatMapID = storedMintData.Data.SeatMapID
		}
	case *seat_mapv4.SeatMapAvailabilityRequest_MintId:
		storedMintData, found := state.GetStore().GetMintResult(identifier.MintId.Value)
		if found {
			seatMapID = storedMintData.SeatMapID
		}
	}

	seatMapAvailability := filterSeatMapAvailabilityByID(mockdata.SeatMapAvailabilityV4, seatMapID)

	if seatMapAvailability == nil {
		return &seat_mapv4.SeatMapAvailabilityResponse{
			Response: &seat_mapv4.SeatMapAvailabilityResponse_ErrorResponse{
				ErrorResponse: &seat_mapv4.SeatMapAvailabilityErrorResponse{
					Header: common.ErrorHeaderV4(typesv4.ErrorCode_ERROR_CODE_INVALID_IDENTIFIERS, "Seat map availability not found for given identifier"),
				},
			},
		}, nil
	}

	return &seat_mapv4.SeatMapAvailabilityResponse{
		Response: &seat_mapv4.SeatMapAvailabilityResponse_SuccessResponse{
			SuccessResponse: &seat_mapv4.SeatMapAvailabilitySuccessResponse{
				Header:  common.SuccessHeaderV4(),
				SeatMap: seatMapAvailability,
			},
		},
	}, nil
}
