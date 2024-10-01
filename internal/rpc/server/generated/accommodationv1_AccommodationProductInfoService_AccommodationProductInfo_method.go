// Code generated by '/home/evdev/Documents/Chain4Travel/Git/camino-messenger-bot/scripts/generate_grpc_service_handlers.sh'. DO NOT EDIT.
// template: templates/server_method.go.tpl

package generated

import (
	"context"
	"fmt"

	accommodationv1 "buf.build/gen/go/chain4travel/camino-messenger-protocol/protocolbuffers/go/cmp/services/accommodation/v1"

	"github.com/chain4travel/camino-messenger-bot/internal/rpc/client/generated"
)

func (s *accommodationv1AccommodationProductInfoServiceSrv) AccommodationProductInfo(ctx context.Context, request *accommodationv1.AccommodationProductInfoRequest) (*accommodationv1.AccommodationProductInfoResponse, error) {
	response, err := s.reqProcessor.ProcessExternalRequest(ctx, generated.AccommodationProductInfoServiceV1Request, request)
	if err != nil {
		return nil, fmt.Errorf("failed to process %s request: %w", generated.AccommodationProductInfoServiceV1Request, err)
	}
	resp, ok := response.(*accommodationv1.AccommodationProductInfoResponse)
	if !ok {
		return nil, fmt.Errorf("invalid response type: expected %s, got %T", generated.AccommodationProductInfoServiceV1Response, response)
	}
	return resp, nil
}
