// Copyright (C) 2022-2025, Chain4Travel AG. All rights reserved.
// See the file LICENSE for licensing terms.

package v1

import (
	"context"
	"fmt"

	"buf.build/gen/go/chain4travel/camino-messenger-protocol/grpc/go/cmp/services/ping/v1/pingv1grpc"
	pingv1 "buf.build/gen/go/chain4travel/camino-messenger-protocol/protocolbuffers/go/cmp/services/ping/v1"
	typesv1 "buf.build/gen/go/chain4travel/camino-messenger-protocol/protocolbuffers/go/cmp/types/v1"
	"github.com/chain4travel/camino-messenger-bot/v11/pp-mock/metadata"
)

var _ pingv1grpc.PingServiceServer = (*pingServiceV1Server)(nil)

type pingServiceV1Server struct{}

func NewPingServiceServer() pingv1grpc.PingServiceServer {
	return &pingServiceV1Server{}
}

func (s *pingServiceV1Server) Ping(ctx context.Context, req *pingv1.PingRequest) (*pingv1.PingResponse, error) {
	md := metadata.FromContext(ctx)
	return &pingv1.PingResponse{
		Header: &typesv1.ResponseHeader{
			Status: typesv1.StatusType_STATUS_TYPE_SUCCESS,
		},
		PingMessage: fmt.Sprintf("Ping response to [%s] with request ID: %s", req.PingMessage, md.RequestID),
	}, nil
}
