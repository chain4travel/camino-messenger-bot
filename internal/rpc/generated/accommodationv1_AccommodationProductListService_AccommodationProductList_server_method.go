// Code generated by 'scripts/generate_grpc_service_handlers.sh'. DO NOT EDIT.
// template: templates/server_method.go.tpl

package generated

import (
	"context"
	"fmt"

	accommodationv1 "buf.build/gen/go/chain4travel/camino-messenger-protocol/protocolbuffers/go/cmp/services/accommodation/v1"
)

func (s *accommodationv1AccommodationProductListServiceServer) AccommodationProductList(ctx context.Context, request *accommodationv1.AccommodationProductListRequest) (*accommodationv1.AccommodationProductListResponse, error) {
	response, err := s.reqProcessor.HandleRequest(ctx, AccommodationProductListServiceV1Request, request)
	if err != nil {
		return nil, fmt.Errorf("failed to process %s request: %w", AccommodationProductListServiceV1Request, err)
	}
	resp, ok := response.(*accommodationv1.AccommodationProductListResponse)
	if !ok {
		return nil, fmt.Errorf("invalid response type: expected %s, got %T", AccommodationProductListServiceV1Response, response)
	}
	return resp, nil
}
