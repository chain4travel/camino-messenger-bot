package v3

import (
	"context"

	"buf.build/gen/go/chain4travel/camino-messenger-protocol/grpc/go/cmp/services/seat_map/v3/seat_mapv3grpc"
	seat_mapv3 "buf.build/gen/go/chain4travel/camino-messenger-protocol/protocolbuffers/go/cmp/services/seat_map/v3"
	typesv1 "buf.build/gen/go/chain4travel/camino-messenger-protocol/protocolbuffers/go/cmp/types/v1"
	mockdata "github.com/chain4travel/camino-messenger-bot/v11/pp-mock/services/data"
)

var _ seat_mapv3grpc.SeatMapServiceServer = (*seatMapV3Server)(nil)

type seatMapV3Server struct{}

func NewSeatMapV3Server() seat_mapv3grpc.SeatMapServiceServer {
	return &seatMapV3Server{}
}

func (s *seatMapV3Server) SeatMap(_ context.Context, req *seat_mapv3.SeatMapRequest) (*seat_mapv3.SeatMapResponse, error) {
	seatMap := filterSeatMapByID(mockdata.SeatMapV3, req.MapId)
	if seatMap == nil {
		return &seat_mapv3.SeatMapResponse{
			Header: &typesv1.ResponseHeader{
				Status: typesv1.StatusType_STATUS_TYPE_FAILURE,
				Alerts: []*typesv1.Alert{{
					Type:    typesv1.AlertType_ALERT_TYPE_ERROR,
					Message: "Seat map not found",
				}},
			},
		}, nil
	}
	seatMap, alerts := filterSeatMapLanguage(seatMap, req.Languages)
	return &seat_mapv3.SeatMapResponse{
		Header: &typesv1.ResponseHeader{
			Status: typesv1.StatusType_STATUS_TYPE_SUCCESS,
			Alerts: alerts,
		},
		SeatMap: seatMap,
	}, nil
}
