// Code generated by 'scripts/generate_grpc_service_handlers.sh'. DO NOT EDIT.
// template: templates/server_method.go.tpl

package generated

import (
	"context"
	"fmt"

	insurancev1 "buf.build/gen/go/chain4travel/camino-messenger-protocol/protocolbuffers/go/cmp/services/insurance/v1"
)

func (s *insurancev1InsuranceProductInfoServiceServer) InsuranceProductInfo(ctx context.Context, request *insurancev1.InsuranceProductInfoRequest) (*insurancev1.InsuranceProductInfoResponse, error) {
	response, err := s.reqProcessor.ProcessExternalRequest(ctx, InsuranceProductInfoServiceV1Request, request)
	if err != nil {
		return nil, fmt.Errorf("failed to process %s request: %w", InsuranceProductInfoServiceV1Request, err)
	}
	resp, ok := response.(*insurancev1.InsuranceProductInfoResponse)
	if !ok {
		return nil, fmt.Errorf("invalid response type: expected %s, got %T", InsuranceProductInfoServiceV1Response, response)
	}
	return resp, nil
}