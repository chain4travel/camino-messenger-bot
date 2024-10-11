// Code generated by 'scripts/generate_grpc_service_handlers.sh'. DO NOT EDIT.
// template: templates/server_method.go.tpl

package generated

import (
	"context"
	"fmt"

	partnerv1 "buf.build/gen/go/chain4travel/camino-messenger-protocol/protocolbuffers/go/cmp/services/partner/v1"
)

func (s *partnerv1GetPartnerConfigurationServiceServer) GetPartnerConfiguration(ctx context.Context, request *partnerv1.GetPartnerConfigurationRequest) (*partnerv1.GetPartnerConfigurationResponse, error) {
	response, err := s.reqProcessor.HandleRequest(ctx, GetPartnerConfigurationServiceV1Request, request)
	if err != nil {
		return nil, fmt.Errorf("failed to process %s request: %w", GetPartnerConfigurationServiceV1Request, err)
	}
	resp, ok := response.(*partnerv1.GetPartnerConfigurationResponse)
	if !ok {
		return nil, fmt.Errorf("invalid response type: expected %s, got %T", GetPartnerConfigurationServiceV1Response, response)
	}
	return resp, nil
}
