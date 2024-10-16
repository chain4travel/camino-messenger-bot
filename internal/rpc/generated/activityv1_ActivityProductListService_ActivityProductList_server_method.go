// Code generated by 'scripts/generate_grpc_service_handlers.sh'. DO NOT EDIT.
// template: templates/server_method.go.tpl

package generated

import (
	"context"
	"fmt"

	activityv1 "buf.build/gen/go/chain4travel/camino-messenger-protocol/protocolbuffers/go/cmp/services/activity/v1"
)

func (s *activityv1ActivityProductListServiceServer) ActivityProductList(ctx context.Context, request *activityv1.ActivityProductListRequest) (*activityv1.ActivityProductListResponse, error) {
	response, err := s.reqProcessor.ProcessExternalRequest(ctx, ActivityProductListServiceV1Request, request)
	if err != nil {
		return nil, fmt.Errorf("failed to process %s request: %w", ActivityProductListServiceV1Request, err)
	}
	resp, ok := response.(*activityv1.ActivityProductListResponse)
	if !ok {
		return nil, fmt.Errorf("invalid response type: expected %s, got %T", ActivityProductListServiceV1Response, response)
	}
	return resp, nil
}
