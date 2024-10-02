// Code generated by 'scripts/generate_grpc_service_handlers.sh'. DO NOT EDIT.
// template: templates/server_method.go.tpl

package generated

import (
	"context"
	"fmt"

	activityv2 "buf.build/gen/go/chain4travel/camino-messenger-protocol/protocolbuffers/go/cmp/services/activity/v2"
)

func (s *activityv2ActivityProductInfoServiceServer) ActivityProductInfo(ctx context.Context, request *activityv2.ActivityProductInfoRequest) (*activityv2.ActivityProductInfoResponse, error) {
	response, err := s.reqProcessor.ProcessExternalRequest(ctx, ActivityProductInfoServiceV2Request, request)
	if err != nil {
		return nil, fmt.Errorf("failed to process %s request: %w", ActivityProductInfoServiceV2Request, err)
	}
	resp, ok := response.(*activityv2.ActivityProductInfoResponse)
	if !ok {
		return nil, fmt.Errorf("invalid response type: expected %s, got %T", ActivityProductInfoServiceV2Response, response)
	}
	return resp, nil
}
