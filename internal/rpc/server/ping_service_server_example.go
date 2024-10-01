package server

import (
	"context"

	"buf.build/gen/go/chain4travel/camino-messenger-protocol/grpc/go/cmp/services/ping/v1/pingv1grpc"
	pingv1 "buf.build/gen/go/chain4travel/camino-messenger-protocol/protocolbuffers/go/cmp/services/ping/v1"

	"github.com/chain4travel/camino-messenger-bot/internal/messaging/messages"
	"google.golang.org/grpc"
)

var (
	_ pingv1grpc.PingServiceServer = (*ping_srv1)(nil)
)

type ping_srv1 struct {
	reqProcessor externalRequestProcessor
}

func NewPingServer(
	grpcServer *grpc.Server,
	reqProcess externalRequestProcessor,
) pingv1grpc.PingServiceServer {
	ping_srv := &ping_srv1{reqProcessor: reqProcess}
	pingv1grpc.RegisterPingServiceServer(grpcServer, ping_srv)
	return ping_srv
}

func (s *ping_srv1) Ping(ctx context.Context, request *pingv1.PingRequest) (*pingv1.PingResponse, error) {
	response, err := s.reqProcessor.processExternalRequest(
		ctx,
		messages.PingRequest, request,
	)
	return response.(*pingv1.PingResponse), err
}
