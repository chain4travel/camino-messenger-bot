// Copyright (C) 2022-2026, Chain4Travel AG. All rights reserved.
// See the file LICENSE for licensing terms.

package v1

import (
	"context"
	"fmt"

	"buf.build/gen/go/chain4travel/camino-messenger-protocol/grpc/go/cmp/services/ping/v1/pingv1grpc"
	pingv1 "buf.build/gen/go/chain4travel/camino-messenger-protocol/protocolbuffers/go/cmp/services/ping/v1"
	"github.com/chain4travel/camino-messenger-bot/v12/pp-mock/common"
	"github.com/chain4travel/camino-messenger-bot/v12/pp-mock/metadata"
)

var _ pingv1grpc.PingServiceServer = (*pingServiceV1Server)(nil)

type pingServiceV1Server struct{}

func NewPingServiceServer() pingv1grpc.PingServiceServer {
	return &pingServiceV1Server{}
}

func (s *pingServiceV1Server) Ping(ctx context.Context, req *pingv1.PingRequest) (*pingv1.PingResponse, error) {
	md := metadata.FromContext(ctx)
	return &pingv1.PingResponse{
		Header:      common.SuccessHeaderV1(),
		PingMessage: fmt.Sprintf("Ping response to [%s] with request ID: %s", req.PingMessage, md.RequestID),
	}, nil
}
