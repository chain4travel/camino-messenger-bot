// Code generated by 'scripts/generate_grpc_service_handlers.sh'. DO NOT EDIT.
// template: templates/server_method.go.tpl

package generated

import (
	"context"
	"fmt"

	infov2 "buf.build/gen/go/chain4travel/camino-messenger-protocol/protocolbuffers/go/cmp/services/info/v2"
)

func (s *infov2CountryEntryRequirementsServiceServer) CountryEntryRequirements(ctx context.Context, request *infov2.CountryEntryRequirementsRequest) (*infov2.CountryEntryRequirementsResponse, error) {
	response, err := s.reqProcessor.ProcessExternalRequest(ctx, CountryEntryRequirementsServiceV2Request, request)
	if err != nil {
		return nil, fmt.Errorf("failed to process %s request: %w", CountryEntryRequirementsServiceV2Request, err)
	}
	resp, ok := response.(*infov2.CountryEntryRequirementsResponse)
	if !ok {
		return nil, fmt.Errorf("invalid response type: expected %s, got %T", CountryEntryRequirementsServiceV2Response, response)
	}
	return resp, nil
}
