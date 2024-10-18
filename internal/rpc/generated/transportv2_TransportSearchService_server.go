// Code generated by './scripts/generate_grpc_service_handlers.sh'. DO NOT EDIT.
// template: templates/server.go.tpl

package generated

import (
	"buf.build/gen/go/chain4travel/camino-messenger-protocol/grpc/go/cmp/services/transport/v2/transportv2grpc"
	"github.com/chain4travel/camino-messenger-bot/internal/rpc"

	"google.golang.org/grpc"
)

var _ transportv2grpc.TransportSearchServiceServer = (*transportv2TransportSearchServiceServer)(nil)

type transportv2TransportSearchServiceServer struct {
	reqProcessor rpc.ExternalRequestProcessor
}

func registerTransportSearchServiceV2Server(grpcServer *grpc.Server, reqProcessor rpc.ExternalRequestProcessor) {
	transportv2grpc.RegisterTransportSearchServiceServer(grpcServer, &transportv2TransportSearchServiceServer{reqProcessor})
}
