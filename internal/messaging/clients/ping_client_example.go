/*
 * Copyright (C) 2022-2023, Chain4Travel AG. All rights reserved.
 * See the file LICENSE for licensing terms.
 */

package clients

import (
	"context"
	"fmt"

	"buf.build/gen/go/chain4travel/camino-messenger-protocol/grpc/go/cmp/services/ping/v1/pingv1grpc"
	pingv1 "buf.build/gen/go/chain4travel/camino-messenger-protocol/protocolbuffers/go/cmp/services/ping/v1"
	"github.com/chain4travel/camino-messenger-bot/internal/messaging/types"

	"google.golang.org/grpc"
	"google.golang.org/protobuf/reflect/protoreflect"
)

const (
	PingServiceV1                           = "cmp.services.ping.v1.PingService"
	PingServiceV1Request  types.MessageType = types.MessageType(PingServiceV1 + ".Request")
	PingServiceV1Response types.MessageType = types.MessageType(PingServiceV1 + ".Response")
)

var _ Client = (*PingClient)(nil)

func NewPingServiceV1(grpcCon *grpc.ClientConn) *PingClient {
	client := pingv1grpc.NewPingServiceClient(grpcCon)
	return &PingClient{client: &client}
}

type PingClient struct {
	client *pingv1grpc.PingServiceClient
}

func (s PingClient) Call(ctx context.Context, requestIntf protoreflect.ProtoMessage, opts ...grpc.CallOption) (protoreflect.ProtoMessage, types.MessageType, error) {
	request, ok := requestIntf.(*pingv1.PingRequest)
	if !ok {
		return nil, PingServiceV1Response, fmt.Errorf("invalid request type")
	}
	response, err := (*s.client).Ping(ctx, request, opts...)
	return response, PingServiceV1Response, err
}
