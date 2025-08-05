package v3

import (
	"context"

	"buf.build/gen/go/chain4travel/camino-messenger-protocol/grpc/go/cmp/services/seat_map/v3/seat_mapv3grpc"
	seat_mapv3 "buf.build/gen/go/chain4travel/camino-messenger-protocol/protocolbuffers/go/cmp/services/seat_map/v3"
	typesv1 "buf.build/gen/go/chain4travel/camino-messenger-protocol/protocolbuffers/go/cmp/types/v1"
	"github.com/chain4travel/camino-messenger-bot/v11/pp-mock/handlers/state"
	mockdata "github.com/chain4travel/camino-messenger-bot/v11/pp-mock/services/data"
)

var _ seat_mapv3grpc.SeatMapAvailabilityServiceServer = (*seatMapAvailabilityV3Server)(nil)

type seatMapAvailabilityV3Server struct{}

func NewSeatMapAvailabilityV3Server() seat_mapv3grpc.SeatMapAvailabilityServiceServer {
	return &seatMapAvailabilityV3Server{}
}

func (s *seatMapAvailabilityV3Server) SeatMapAvailability(_ context.Context, req *seat_mapv3.SeatMapAvailabilityRequest) (*seat_mapv3.SeatMapAvailabilityResponse, error) {
	seatMapIndex := -1
	switch identifier := req.Identifier.(type) {
	case *seat_mapv3.SeatMapAvailabilityRequest_SearchIdentifier:
		storedMintData, found := state.GetStore().GetSearchResult(identifier.SearchIdentifier.SearchId.Value)
		if found {
			seatMapIndex = storedMintData.Data.SeatMapIndex
		}
	case *seat_mapv3.SeatMapAvailabilityRequest_MintId:
		storedMintData, found := state.GetStore().GetMintResult(identifier.MintId)
		if found {
			seatMapIndex = storedMintData.SeatMapIndex
		}
	}

	if seatMapIndex == -1 {
		return &seat_mapv3.SeatMapAvailabilityResponse{
			Header: &typesv1.ResponseHeader{
				Status: typesv1.StatusType_STATUS_TYPE_FAILURE,
				Alerts: []*typesv1.Alert{{
					Type:    typesv1.AlertType_ALERT_TYPE_ERROR,
					Message: "Seat map availability not found for given identifier",
				}},
			},
		}, nil
	}

	return &seat_mapv3.SeatMapAvailabilityResponse{
		Header: &typesv1.ResponseHeader{
			Status: typesv1.StatusType_STATUS_TYPE_SUCCESS,
		},
		SeatMap: mockdata.SeatMapAvailabilityV3[seatMapIndex],
	}, nil
}
