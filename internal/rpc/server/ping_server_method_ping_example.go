package server

import (
	"context"
	"fmt"

	pingv1 "buf.build/gen/go/chain4travel/camino-messenger-protocol/protocolbuffers/go/cmp/services/ping/v1"

	"github.com/chain4travel/camino-messenger-bot/internal/messaging/clients"
)

func (s *ping_srv1) Ping(ctx context.Context, request *pingv1.PingRequest) (*pingv1.PingResponse, error) {
	response, err := s.reqProcessor.processExternalRequest(ctx, clients.PingServiceV1Request, request)
	if err != nil {
		return nil, fmt.Errorf("failed to process %s request: %w", clients.PingServiceV1Request, err)
	}
	pingResp, ok := response.(*pingv1.PingResponse)
	if !ok {
		return nil, fmt.Errorf("invalid response type: expected %s, got %T", clients.PingServiceV1Response, response)
	}
	return pingResp, nil
}
