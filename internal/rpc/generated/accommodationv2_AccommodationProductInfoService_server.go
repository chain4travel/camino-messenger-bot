// Code generated by 'scripts/generate_grpc_service_handlers.sh'. DO NOT EDIT.
// template: templates/server.go.tpl

package generated

import (
	"buf.build/gen/go/chain4travel/camino-messenger-protocol/grpc/go/cmp/services/accommodation/v2/accommodationv2grpc"
	"github.com/chain4travel/camino-messenger-bot/internal/rpc"

	"google.golang.org/grpc"
)

var _ accommodationv2grpc.AccommodationProductInfoServiceServer = (*accommodationv2AccommodationProductInfoServiceServer)(nil)

type accommodationv2AccommodationProductInfoServiceServer struct {
	reqProcessor rpc.ExternalRequestProcessor
}

func registerAccommodationProductInfoServiceV2Server(grpcServer *grpc.Server, reqProcessor rpc.ExternalRequestProcessor) {
	accommodationv2grpc.RegisterAccommodationProductInfoServiceServer(grpcServer, &accommodationv2AccommodationProductInfoServiceServer{reqProcessor})
}