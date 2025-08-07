// Copyright (C) 2022-2025, Chain4Travel AG. All rights reserved.
// See the file LICENSE for licensing terms.

package v4

import (
	"context"

	"buf.build/gen/go/chain4travel/camino-messenger-protocol/grpc/go/cmp/services/seat_map/v4/seat_mapv4grpc"
	seat_mapv4 "buf.build/gen/go/chain4travel/camino-messenger-protocol/protocolbuffers/go/cmp/services/seat_map/v4"
	"github.com/chain4travel/camino-messenger-bot/v11/pp-mock/common"
	"github.com/chain4travel/camino-messenger-bot/v11/pp-mock/handlers/state"
	mockdata "github.com/chain4travel/camino-messenger-bot/v11/pp-mock/services/data"
)

var _ seat_mapv4grpc.SeatMapAvailabilityServiceServer = (*seatMapAvailabilityV4Server)(nil)

type seatMapAvailabilityV4Server struct{}

func NewSeatMapAvailabilityServer() seat_mapv4grpc.SeatMapAvailabilityServiceServer {
	return &seatMapAvailabilityV4Server{}
}

func (s *seatMapAvailabilityV4Server) SeatMapAvailability(_ context.Context, req *seat_mapv4.SeatMapAvailabilityRequest) (*seat_mapv4.SeatMapAvailabilityResponse, error) {
	seatMapIndex := -1
	switch identifier := req.Identifier.(type) {
	case *seat_mapv4.SeatMapAvailabilityRequest_SearchIdentifier:
		storedMintData, found := state.GetStore().GetSearchResult(identifier.SearchIdentifier.SearchId.Value)
		if found {
			seatMapIndex = storedMintData.Data.SeatMapIndex
		}
	case *seat_mapv4.SeatMapAvailabilityRequest_MintId:
		storedMintData, found := state.GetStore().GetMintResult(identifier.MintId.Value)
		if found {
			seatMapIndex = storedMintData.SeatMapIndex
		}
	}

	resp := &seat_mapv4.SeatMapAvailabilityResponse{Header: common.SuccessHeaderV4()}

	if seatMapIndex == -1 {
		common.AddHeaderErrorV4(resp.Header, "Seat map availability not found for given identifier")
		return resp, nil
	}

	resp.SeatMap = mockdata.SeatMapAvailabilityV4[seatMapIndex]
	return resp, nil
}
