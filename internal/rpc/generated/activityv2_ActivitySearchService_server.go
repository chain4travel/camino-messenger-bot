// Code generated by 'scripts/generate_grpc_service_handlers.sh'. DO NOT EDIT.
// template: templates/server.go.tpl

package generated

import (
	"buf.build/gen/go/chain4travel/camino-messenger-protocol/grpc/go/cmp/services/activity/v2/activityv2grpc"
	"github.com/chain4travel/camino-messenger-bot/internal/rpc"

	"google.golang.org/grpc"
)

var _ activityv2grpc.ActivitySearchServiceServer = (*activityv2ActivitySearchServiceServer)(nil)

type activityv2ActivitySearchServiceServer struct {
	reqProcessor rpc.ExternalRequestProcessor
}

func registerActivitySearchServiceV2Server(grpcServer *grpc.Server, reqProcessor rpc.ExternalRequestProcessor) {
	activityv2grpc.RegisterActivitySearchServiceServer(grpcServer, &activityv2ActivitySearchServiceServer{reqProcessor})
}