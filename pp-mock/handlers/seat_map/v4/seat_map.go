package v3

import (
	"context"

	"buf.build/gen/go/chain4travel/camino-messenger-protocol/grpc/go/cmp/services/seat_map/v4/seat_mapv4grpc"
	seat_mapv4 "buf.build/gen/go/chain4travel/camino-messenger-protocol/protocolbuffers/go/cmp/services/seat_map/v4"
	typesv4 "buf.build/gen/go/chain4travel/camino-messenger-protocol/protocolbuffers/go/cmp/types/v4"
)

var _ seat_mapv4grpc.SeatMapServiceServer = (*seatMapV4Server)(nil)

type seatMapV4Server struct{}

func NewSeatMapServer() seat_mapv4grpc.SeatMapServiceServer {
	return &seatMapV4Server{}
}

func (s *seatMapV4Server) SeatMap(_ context.Context, req *seat_mapv4.SeatMapRequest) (*seat_mapv4.SeatMapResponse, error) {
	// seatMap := filterSeatMapByID(mockdata.SeatMapV4, req.MapId)
	// if seatMap == nil {
	return &seat_mapv4.SeatMapResponse{
		Header: &typesv4.ResponseHeader{
			Status: typesv4.StatusType_STATUS_TYPE_FAILURE,
			Alerts: []*typesv4.Alert{{
				Type:    typesv4.AlertType_ALERT_TYPE_ERROR,
				Message: "Seat map not found",
			}},
		},
	}, nil
	// }
	// seatMap, alerts := filterSeatMapLanguage(seatMap, req.Languages)
	// return &seat_mapv4.SeatMapResponse{
	// 	Header: &typesv4.ResponseHeader{
	// 		Status: typesv4.StatusType_STATUS_TYPE_SUCCESS,
	// 		Alerts: alerts,
	// 	},
	// 	SeatMap: seatMap,
	// }, nil
}
