// Code generated by './scripts/generate_grpc_service_handlers.sh'. DO NOT EDIT.
// template: templates/server_method.go.tpl

package generated

import (
	"context"
	"fmt"

	accommodationv2 "buf.build/gen/go/chain4travel/camino-messenger-protocol/protocolbuffers/go/cmp/services/accommodation/v2"
)

func (s *accommodationv2AccommodationProductListServiceServer) AccommodationProductList(ctx context.Context, request *accommodationv2.AccommodationProductListRequest) (*accommodationv2.AccommodationProductListResponse, error) {
	response, err := s.reqProcessor.ProcessExternalRequest(ctx, AccommodationProductListServiceV2Request, request)
	if err != nil {
		return nil, fmt.Errorf("failed to process %s request: %w", AccommodationProductListServiceV2Request, err)
	}
	resp, ok := response.(*accommodationv2.AccommodationProductListResponse)
	if !ok {
		return nil, fmt.Errorf("invalid response type: expected %s, got %T", AccommodationProductListServiceV2Response, response)
	}
	return resp, nil
}
