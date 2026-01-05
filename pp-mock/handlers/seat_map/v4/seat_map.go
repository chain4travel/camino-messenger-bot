// Copyright (C) 2022-2026, Chain4Travel AG. All rights reserved.
// See the file LICENSE for licensing terms.

package v4

import (
	"context"

	"buf.build/gen/go/chain4travel/camino-messenger-protocol/grpc/go/cmp/services/seat_map/v4/seat_mapv4grpc"
	seat_mapv4 "buf.build/gen/go/chain4travel/camino-messenger-protocol/protocolbuffers/go/cmp/services/seat_map/v4"
	typesv4 "buf.build/gen/go/chain4travel/camino-messenger-protocol/protocolbuffers/go/cmp/types/v4"
	"github.com/chain4travel/camino-messenger-bot/v12/pp-mock/common"
	mockdata "github.com/chain4travel/camino-messenger-bot/v12/pp-mock/services/data"
)

var _ seat_mapv4grpc.SeatMapServiceServer = (*seatMapV4Server)(nil)

type seatMapV4Server struct{}

func NewSeatMapServer() seat_mapv4grpc.SeatMapServiceServer {
	return &seatMapV4Server{}
}

func (s *seatMapV4Server) SeatMap(_ context.Context, req *seat_mapv4.SeatMapRequest) (*seat_mapv4.SeatMapResponse, error) {
	seatMap := filterSeatMapByID(mockdata.SeatMapV4, req.MapId)

	if seatMap == nil {
		return &seat_mapv4.SeatMapResponse{
			Response: &seat_mapv4.SeatMapResponse_ErrorResponse{
				ErrorResponse: &seat_mapv4.SeatMapErrorResponse{
					Header: common.ErrorHeaderV4(typesv4.ErrorCode_ERROR_CODE_INVALID_IDENTIFIERS, "Seat map not found"),
				},
			},
		}, nil
	}

	seatMap, missingLocalization := filterSeatMapLocalization(seatMap, req.Languages)

	resp := &seat_mapv4.SeatMapResponse{
		Response: &seat_mapv4.SeatMapResponse_SuccessResponse{
			SuccessResponse: &seat_mapv4.SeatMapSuccessResponse{
				Header:  common.SuccessHeaderV4(),
				SeatMap: seatMap,
			},
		},
	}

	if missingLocalization {
		common.AddHeaderAlertV4(resp.GetSuccessResponse().Header, typesv4.AlertCode_ALERT_CODE_INFORMATIONAL, "Seat map is missing localized string for requested languages")
	}

	return resp, nil
}
