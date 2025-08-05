package v3

import (
	"context"

	"buf.build/gen/go/chain4travel/camino-messenger-protocol/grpc/go/cmp/services/seat_map/v4/seat_mapv4grpc"
	seat_mapv4 "buf.build/gen/go/chain4travel/camino-messenger-protocol/protocolbuffers/go/cmp/services/seat_map/v4"
	typesv4 "buf.build/gen/go/chain4travel/camino-messenger-protocol/protocolbuffers/go/cmp/types/v4"
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

	if seatMapIndex == -1 {
		return &seat_mapv4.SeatMapAvailabilityResponse{
			Header: &typesv4.ResponseHeader{
				Status: typesv4.StatusType_STATUS_TYPE_FAILURE,
				Alerts: []*typesv4.Alert{{
					Type:    typesv4.AlertType_ALERT_TYPE_ERROR,
					Message: "Seat map availability not found for given identifier",
				}},
			},
		}, nil
	}

	return &seat_mapv4.SeatMapAvailabilityResponse{
		Header: &typesv4.ResponseHeader{
			Status: typesv4.StatusType_STATUS_TYPE_SUCCESS,
		},
		SeatMap: mockdata.SeatMapAvailabilityV4[seatMapIndex],
	}, nil
}
