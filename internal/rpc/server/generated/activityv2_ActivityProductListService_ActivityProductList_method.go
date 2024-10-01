// Code generated by '/home/evdev/Documents/Chain4Travel/Git/camino-messenger-bot/scripts/generate_grpc_service_handlers.sh'. DO NOT EDIT.
// template: templates/server_method.go.tpl

package generated

import (
	"context"
	"fmt"

	activityv2 "buf.build/gen/go/chain4travel/camino-messenger-protocol/protocolbuffers/go/cmp/services/activity/v2"

	"github.com/chain4travel/camino-messenger-bot/internal/rpc/client/generated"
)

func (s *activityv2ActivityProductListServiceSrv) ActivityProductList(ctx context.Context, request *activityv2.ActivityProductListRequest) (*activityv2.ActivityProductListResponse, error) {
	response, err := s.reqProcessor.ProcessExternalRequest(ctx, generated.ActivityProductListServiceV2Request, request)
	if err != nil {
		return nil, fmt.Errorf("failed to process %s request: %w", generated.ActivityProductListServiceV2Request, err)
	}
	resp, ok := response.(*activityv2.ActivityProductListResponse)
	if !ok {
		return nil, fmt.Errorf("invalid response type: expected %s, got %T", generated.ActivityProductListServiceV2Response, response)
	}
	return resp, nil
}
