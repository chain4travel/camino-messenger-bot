// Code generated by 'scripts/generate_grpc_service_handlers.sh'. DO NOT EDIT.
// template: templates/server_method.go.tpl

package generated

import (
	"context"
	"fmt"

	accommodationv2 "buf.build/gen/go/chain4travel/camino-messenger-protocol/protocolbuffers/go/cmp/services/accommodation/v2"
)

func (s *accommodationv2AccommodationProductInfoServiceServer) AccommodationProductInfo(ctx context.Context, request *accommodationv2.AccommodationProductInfoRequest) (*accommodationv2.AccommodationProductInfoResponse, error) {
	response, err := s.reqProcessor.ProcessExternalRequest(ctx, AccommodationProductInfoServiceV2Request, request)
	if err != nil {
		return nil, fmt.Errorf("failed to process %s request: %w", AccommodationProductInfoServiceV2Request, err)
	}
	resp, ok := response.(*accommodationv2.AccommodationProductInfoResponse)
	if !ok {
		return nil, fmt.Errorf("invalid response type: expected %s, got %T", AccommodationProductInfoServiceV2Response, response)
	}
	return resp, nil
}