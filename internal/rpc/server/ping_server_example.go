package server

import (
	"buf.build/gen/go/chain4travel/camino-messenger-protocol/grpc/go/cmp/services/ping/v1/pingv1grpc"

	"google.golang.org/grpc"
)

var _ pingv1grpc.PingServiceServer = (*ping_srv1)(nil)

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
