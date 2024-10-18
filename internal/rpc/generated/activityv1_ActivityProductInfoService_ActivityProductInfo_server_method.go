// Code generated by './scripts/generate_grpc_service_handlers.sh'. DO NOT EDIT.
// template: templates/server_method.go.tpl

package generated

import (
	"context"
	"fmt"

	activityv1 "buf.build/gen/go/chain4travel/camino-messenger-protocol/protocolbuffers/go/cmp/services/activity/v1"
)

func (s *activityv1ActivityProductInfoServiceServer) ActivityProductInfo(ctx context.Context, request *activityv1.ActivityProductInfoRequest) (*activityv1.ActivityProductInfoResponse, error) {
	response, err := s.reqProcessor.ProcessExternalRequest(ctx, ActivityProductInfoServiceV1Request, request)
	if err != nil {
		return nil, fmt.Errorf("failed to process %s request: %w", ActivityProductInfoServiceV1Request, err)
	}
	resp, ok := response.(*activityv1.ActivityProductInfoResponse)
	if !ok {
		return nil, fmt.Errorf("invalid response type: expected %s, got %T", ActivityProductInfoServiceV1Response, response)
	}
	return resp, nil
}
