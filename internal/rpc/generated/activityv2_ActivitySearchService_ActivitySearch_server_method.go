// Code generated by 'scripts/generate_grpc_service_handlers.sh'. DO NOT EDIT.
// template: templates/server_method.go.tpl

package generated

import (
	"context"
	"fmt"

	activityv2 "buf.build/gen/go/chain4travel/camino-messenger-protocol/protocolbuffers/go/cmp/services/activity/v2"
)

func (s *activityv2ActivitySearchServiceServer) ActivitySearch(ctx context.Context, request *activityv2.ActivitySearchRequest) (*activityv2.ActivitySearchResponse, error) {
	response, err := s.reqProcessor.ProcessExternalRequest(ctx, ActivitySearchServiceV2Request, request)
	if err != nil {
		return nil, fmt.Errorf("failed to process %s request: %w", ActivitySearchServiceV2Request, err)
	}
	resp, ok := response.(*activityv2.ActivitySearchResponse)
	if !ok {
		return nil, fmt.Errorf("invalid response type: expected %s, got %T", ActivitySearchServiceV2Response, response)
	}
	return resp, nil
}
