// Code generated by './scripts/generate_grpc_service_handlers.sh'. DO NOT EDIT.
// template: templates/server.go.tpl

package generated

import (
	"buf.build/gen/go/chain4travel/camino-messenger-protocol/grpc/go/cmp/services/network/v1/networkv1grpc"
	"github.com/chain4travel/camino-messenger-bot/internal/rpc"

	"google.golang.org/grpc"
)

var _ networkv1grpc.GetNetworkFeeServiceServer = (*networkv1GetNetworkFeeServiceServer)(nil)

type networkv1GetNetworkFeeServiceServer struct {
	reqProcessor rpc.ExternalRequestProcessor
}

func registerGetNetworkFeeServiceV1Server(grpcServer *grpc.Server, reqProcessor rpc.ExternalRequestProcessor) {
	networkv1grpc.RegisterGetNetworkFeeServiceServer(grpcServer, &networkv1GetNetworkFeeServiceServer{reqProcessor})
}
