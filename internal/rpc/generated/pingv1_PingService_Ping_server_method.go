// Code generated by 'scripts/generate_grpc_service_handlers.sh'. DO NOT EDIT.
// template: templates/server_method.go.tpl

package generated

import (
	"context"
	"fmt"

	pingv1 "buf.build/gen/go/chain4travel/camino-messenger-protocol/protocolbuffers/go/cmp/services/ping/v1"
)

func (s *pingv1PingServiceServer) Ping(ctx context.Context, request *pingv1.PingRequest) (*pingv1.PingResponse, error) {
	response, err := s.reqProcessor.ProcessExternalRequest(ctx, PingServiceV1Request, request)
	if err != nil {
		return nil, fmt.Errorf("failed to process %s request: %w", PingServiceV1Request, err)
	}
	resp, ok := response.(*pingv1.PingResponse)
	if !ok {
		return nil, fmt.Errorf("invalid response type: expected %s, got %T", PingServiceV1Response, response)
	}
	return resp, nil
}
