// Code generated by './scripts/generate_grpc_service_handlers.sh'. DO NOT EDIT.
// template: templates/server.go.tpl

package generated

import (
	"buf.build/gen/go/chain4travel/camino-messenger-protocol/grpc/go/cmp/services/seat_map/v1/seat_mapv1grpc"
	"github.com/chain4travel/camino-messenger-bot/internal/rpc"

	"google.golang.org/grpc"
)

var _ seat_mapv1grpc.SeatMapServiceServer = (*seat_mapv1SeatMapServiceServer)(nil)

type seat_mapv1SeatMapServiceServer struct {
	reqProcessor rpc.ExternalRequestProcessor
}

func registerSeatMapServiceV1Server(grpcServer *grpc.Server, reqProcessor rpc.ExternalRequestProcessor) {
	seat_mapv1grpc.RegisterSeatMapServiceServer(grpcServer, &seat_mapv1SeatMapServiceServer{reqProcessor})
}
