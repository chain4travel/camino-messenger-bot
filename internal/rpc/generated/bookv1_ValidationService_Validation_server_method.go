// Code generated by 'scripts/generate_grpc_service_handlers.sh'. DO NOT EDIT.
// template: templates/server_method.go.tpl

package generated

import (
	"context"
	"fmt"

	bookv1 "buf.build/gen/go/chain4travel/camino-messenger-protocol/protocolbuffers/go/cmp/services/book/v1"
)

func (s *bookv1ValidationServiceServer) Validation(ctx context.Context, request *bookv1.ValidationRequest) (*bookv1.ValidationResponse, error) {
	response, err := s.reqProcessor.ProcessExternalRequest(ctx, ValidationServiceV1Request, request)
	if err != nil {
		return nil, fmt.Errorf("failed to process %s request: %w", ValidationServiceV1Request, err)
	}
	resp, ok := response.(*bookv1.ValidationResponse)
	if !ok {
		return nil, fmt.Errorf("invalid response type: expected %s, got %T", ValidationServiceV1Response, response)
	}
	return resp, nil
}
