// Code generated by './scripts/generate_grpc_service_handlers.sh'. DO NOT EDIT.
// template: templates/server.go.tpl

package generated

import (
	"buf.build/gen/go/chain4travel/camino-messenger-protocol/grpc/go/cmp/services/accommodation/v2/accommodationv2grpc"
	"github.com/chain4travel/camino-messenger-bot/internal/rpc"

	"google.golang.org/grpc"
)

var _ accommodationv2grpc.AccommodationProductListServiceServer = (*accommodationv2AccommodationProductListServiceServer)(nil)

type accommodationv2AccommodationProductListServiceServer struct {
	reqProcessor rpc.ExternalRequestProcessor
}

func registerAccommodationProductListServiceV2Server(grpcServer *grpc.Server, reqProcessor rpc.ExternalRequestProcessor) {
	accommodationv2grpc.RegisterAccommodationProductListServiceServer(grpcServer, &accommodationv2AccommodationProductListServiceServer{reqProcessor})
}
