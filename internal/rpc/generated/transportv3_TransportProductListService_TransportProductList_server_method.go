// Code generated by './scripts/generate_grpc_service_handlers.sh'. DO NOT EDIT.
// template: templates/server_method.go.tpl

package generated

import (
	"context"
	"fmt"

	transportv3 "buf.build/gen/go/chain4travel/camino-messenger-protocol/protocolbuffers/go/cmp/services/transport/v3"
)

func (s *transportv3TransportProductListServiceServer) TransportProductList(ctx context.Context, request *transportv3.TransportProductListRequest) (*transportv3.TransportProductListResponse, error) {
	response, err := s.reqProcessor.ProcessExternalRequest(ctx, TransportProductListServiceV3Request, request)
	if err != nil {
		return nil, fmt.Errorf("failed to process %s request: %w", TransportProductListServiceV3Request, err)
	}
	resp, ok := response.(*transportv3.TransportProductListResponse)
	if !ok {
		return nil, fmt.Errorf("invalid response type: expected %s, got %T", TransportProductListServiceV3Response, response)
	}
	return resp, nil
}
