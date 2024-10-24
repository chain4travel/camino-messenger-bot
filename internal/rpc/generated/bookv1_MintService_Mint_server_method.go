// Code generated by 'scripts/generate_grpc_service_handlers.sh'. DO NOT EDIT.
// template: templates/server_method.go.tpl

package generated

import (
	"context"
	"fmt"

	bookv1 "buf.build/gen/go/chain4travel/camino-messenger-protocol/protocolbuffers/go/cmp/services/book/v1"
)

func (s *bookv1MintServiceServer) Mint(ctx context.Context, request *bookv1.MintRequest) (*bookv1.MintResponse, error) {
	response, err := s.reqProcessor.ProcessExternalRequest(ctx, MintServiceV1Request, request)
	if err != nil {
		return nil, fmt.Errorf("failed to process %s request: %w", MintServiceV1Request, err)
	}
	resp, ok := response.(*bookv1.MintResponse)
	if !ok {
		return nil, fmt.Errorf("invalid response type: expected %s, got %T", MintServiceV1Response, response)
	}
	return resp, nil
}
