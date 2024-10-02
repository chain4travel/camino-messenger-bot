// Code generated by 'scripts/generate_grpc_service_handlers.sh'. DO NOT EDIT.
// template: templates/server_method.go.tpl

package generated

import (
	"context"
	"fmt"

	transportv1 "buf.build/gen/go/chain4travel/camino-messenger-protocol/protocolbuffers/go/cmp/services/transport/v1"
)

func (s *transportv1TransportSearchServiceServer) TransportSearch(ctx context.Context, request *transportv1.TransportSearchRequest) (*transportv1.TransportSearchResponse, error) {
	response, err := s.reqProcessor.ProcessExternalRequest(ctx, TransportSearchServiceV1Request, request)
	if err != nil {
		return nil, fmt.Errorf("failed to process %s request: %w", TransportSearchServiceV1Request, err)
	}
	resp, ok := response.(*transportv1.TransportSearchResponse)
	if !ok {
		return nil, fmt.Errorf("invalid response type: expected %s, got %T", TransportSearchServiceV1Response, response)
	}
	return resp, nil
}