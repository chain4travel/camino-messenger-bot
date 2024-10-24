// Code generated by 'scripts/generate_grpc_service_handlers.sh'. DO NOT EDIT.
// template: templates/server_method.go.tpl

package generated

import (
	"context"
	"fmt"

	infov1 "buf.build/gen/go/chain4travel/camino-messenger-protocol/protocolbuffers/go/cmp/services/info/v1"
)

func (s *infov1CountryEntryRequirementsServiceServer) CountryEntryRequirements(ctx context.Context, request *infov1.CountryEntryRequirementsRequest) (*infov1.CountryEntryRequirementsResponse, error) {
	response, err := s.reqProcessor.ProcessExternalRequest(ctx, CountryEntryRequirementsServiceV1Request, request)
	if err != nil {
		return nil, fmt.Errorf("failed to process %s request: %w", CountryEntryRequirementsServiceV1Request, err)
	}
	resp, ok := response.(*infov1.CountryEntryRequirementsResponse)
	if !ok {
		return nil, fmt.Errorf("invalid response type: expected %s, got %T", CountryEntryRequirementsServiceV1Response, response)
	}
	return resp, nil
}
