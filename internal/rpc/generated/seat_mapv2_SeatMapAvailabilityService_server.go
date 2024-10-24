// Code generated by 'scripts/generate_grpc_service_handlers.sh'. DO NOT EDIT.
// template: templates/server.go.tpl

package generated

import (
	"buf.build/gen/go/chain4travel/camino-messenger-protocol/grpc/go/cmp/services/seat_map/v2/seat_mapv2grpc"
	"github.com/chain4travel/camino-messenger-bot/internal/rpc"

	"google.golang.org/grpc"
)

var _ seat_mapv2grpc.SeatMapAvailabilityServiceServer = (*seat_mapv2SeatMapAvailabilityServiceServer)(nil)

type seat_mapv2SeatMapAvailabilityServiceServer struct {
	reqProcessor rpc.ExternalRequestProcessor
}

func registerSeatMapAvailabilityServiceV2Server(grpcServer *grpc.Server, reqProcessor rpc.ExternalRequestProcessor) {
	seat_mapv2grpc.RegisterSeatMapAvailabilityServiceServer(grpcServer, &seat_mapv2SeatMapAvailabilityServiceServer{reqProcessor})
}
