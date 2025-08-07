// Copyright (C) 2022-2025, Chain4Travel AG. All rights reserved.
// See the file LICENSE for licensing terms.

package v4

import (
	"context"

	"buf.build/gen/go/chain4travel/camino-messenger-protocol/grpc/go/cmp/services/seat_map/v4/seat_mapv4grpc"
	seat_mapv4 "buf.build/gen/go/chain4travel/camino-messenger-protocol/protocolbuffers/go/cmp/services/seat_map/v4"
	"github.com/chain4travel/camino-messenger-bot/v11/pp-mock/common"
	mockdata "github.com/chain4travel/camino-messenger-bot/v11/pp-mock/services/data"
)

var _ seat_mapv4grpc.SeatMapServiceServer = (*seatMapV4Server)(nil)

type seatMapV4Server struct{}

func NewSeatMapServer() seat_mapv4grpc.SeatMapServiceServer {
	return &seatMapV4Server{}
}

func (s *seatMapV4Server) SeatMap(_ context.Context, req *seat_mapv4.SeatMapRequest) (*seat_mapv4.SeatMapResponse, error) {
	seatMap := filterSeatMapByID(mockdata.SeatMapV4, req.MapId)

	resp := &seat_mapv4.SeatMapResponse{Header: common.SuccessHeaderV4()}

	if seatMap == nil {
		common.AddHeaderErrorV4(resp.Header, "Seat map not found")
		return resp, nil
	}

	seatMap, missingLocalization := filterSeatMapLocalization(seatMap, req.Languages)

	if missingLocalization {
		common.AddHeaderWarningV4(resp.Header, "Seat map is missing localized string for requested languages")
	}

	resp.SeatMap = seatMap
	return resp, nil
}
