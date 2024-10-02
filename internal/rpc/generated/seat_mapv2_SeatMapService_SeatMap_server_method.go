// Code generated by 'scripts/generate_grpc_service_handlers.sh'. DO NOT EDIT.
// template: templates/server_method.go.tpl

package generated

import (
	"context"
	"fmt"

	seat_mapv2 "buf.build/gen/go/chain4travel/camino-messenger-protocol/protocolbuffers/go/cmp/services/seat_map/v2"
)

func (s *seat_mapv2SeatMapServiceServer) SeatMap(ctx context.Context, request *seat_mapv2.SeatMapRequest) (*seat_mapv2.SeatMapResponse, error) {
	response, err := s.reqProcessor.ProcessExternalRequest(ctx, SeatMapServiceV2Request, request)
	if err != nil {
		return nil, fmt.Errorf("failed to process %s request: %w", SeatMapServiceV2Request, err)
	}
	resp, ok := response.(*seat_mapv2.SeatMapResponse)
	if !ok {
		return nil, fmt.Errorf("invalid response type: expected %s, got %T", SeatMapServiceV2Response, response)
	}
	return resp, nil
}