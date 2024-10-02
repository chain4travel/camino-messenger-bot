// Code generated by 'scripts/generate_grpc_service_handlers.sh'. DO NOT EDIT.
// template: templates/client.go.tpl

package generated

import (
	"buf.build/gen/go/chain4travel/camino-messenger-protocol/grpc/go/cmp/services/seat_map/v2/seat_mapv2grpc"
	"github.com/chain4travel/camino-messenger-bot/internal/messaging/types"
	"github.com/chain4travel/camino-messenger-bot/internal/rpc"

	"google.golang.org/grpc"
)

const (
	SeatMapServiceV2                           = "cmp.services.seat_map.v2.SeatMapService"
	SeatMapServiceV2Request  types.MessageType = types.MessageType(SeatMapServiceV2 + ".Request")
	SeatMapServiceV2Response types.MessageType = types.MessageType(SeatMapServiceV2 + ".Response")
)

var _ rpc.Client = (*SeatMapServiceV2Client)(nil)

func NewSeatMapServiceV2(grpcCon *grpc.ClientConn) *SeatMapServiceV2Client {
	client := seat_mapv2grpc.NewSeatMapServiceClient(grpcCon)
	return &SeatMapServiceV2Client{client: &client}
}

type SeatMapServiceV2Client struct {
	client *seat_mapv2grpc.SeatMapServiceClient
}