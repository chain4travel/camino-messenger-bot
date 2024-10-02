// Code generated by 'scripts/generate_grpc_service_handlers.sh'. DO NOT EDIT.
// template: templates/server.go.tpl

package generated

import (
	"buf.build/gen/go/chain4travel/camino-messenger-protocol/grpc/go/cmp/services/accommodation/v1/accommodationv1grpc"
	"github.com/chain4travel/camino-messenger-bot/internal/rpc"

	"google.golang.org/grpc"
)

var _ accommodationv1grpc.AccommodationProductListServiceServer = (*accommodationv1AccommodationProductListServiceServer)(nil)

type accommodationv1AccommodationProductListServiceServer struct {
	reqProcessor rpc.ExternalRequestProcessor
}

func registerAccommodationProductListServiceV1Server(grpcServer *grpc.Server, reqProcessor rpc.ExternalRequestProcessor) {
	accommodationv1grpc.RegisterAccommodationProductListServiceServer(grpcServer, &accommodationv1AccommodationProductListServiceServer{reqProcessor})
}
