// Code generated by 'scripts/generate_grpc_service_handlers.sh'. DO NOT EDIT.
// template: templates/server_method.go.tpl

package generated

import (
	"context"
	"fmt"

	seat_mapv1 "buf.build/gen/go/chain4travel/camino-messenger-protocol/protocolbuffers/go/cmp/services/seat_map/v1"
)

func (s *seat_mapv1SeatMapAvailabilityServiceServer) SeatMapAvailability(ctx context.Context, request *seat_mapv1.SeatMapAvailabilityRequest) (*seat_mapv1.SeatMapAvailabilityResponse, error) {
	response, err := s.reqProcessor.ProcessExternalRequest(ctx, SeatMapAvailabilityServiceV1Request, request)
	if err != nil {
		return nil, fmt.Errorf("failed to process %s request: %w", SeatMapAvailabilityServiceV1Request, err)
	}
	resp, ok := response.(*seat_mapv1.SeatMapAvailabilityResponse)
	if !ok {
		return nil, fmt.Errorf("invalid response type: expected %s, got %T", SeatMapAvailabilityServiceV1Response, response)
	}
	return resp, nil
}
