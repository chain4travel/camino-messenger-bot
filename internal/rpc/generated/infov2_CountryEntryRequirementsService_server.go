// Code generated by 'scripts/generate_grpc_service_handlers.sh'. DO NOT EDIT.
// template: templates/server.go.tpl

package generated

import (
	"buf.build/gen/go/chain4travel/camino-messenger-protocol/grpc/go/cmp/services/info/v2/infov2grpc"
	"github.com/chain4travel/camino-messenger-bot/internal/rpc"

	"google.golang.org/grpc"
)

var _ infov2grpc.CountryEntryRequirementsServiceServer = (*infov2CountryEntryRequirementsServiceServer)(nil)

type infov2CountryEntryRequirementsServiceServer struct {
	reqProcessor rpc.ExternalRequestProcessor
}

func registerCountryEntryRequirementsServiceV2Server(grpcServer *grpc.Server, reqProcessor rpc.ExternalRequestProcessor) {
	infov2grpc.RegisterCountryEntryRequirementsServiceServer(grpcServer, &infov2CountryEntryRequirementsServiceServer{reqProcessor})
}