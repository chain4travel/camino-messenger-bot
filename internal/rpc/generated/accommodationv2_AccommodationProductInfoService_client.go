// Code generated by './scripts/generate_grpc_service_handlers.sh'. DO NOT EDIT.
// template: templates/client.go.tpl

package generated

import (
	"buf.build/gen/go/chain4travel/camino-messenger-protocol/grpc/go/cmp/services/accommodation/v2/accommodationv2grpc"
	"github.com/chain4travel/camino-messenger-bot/internal/messaging/types"
	"github.com/chain4travel/camino-messenger-bot/internal/rpc"

	"google.golang.org/grpc"
)

const (
	AccommodationProductInfoServiceV2                           = "cmp.services.accommodation.v2.AccommodationProductInfoService"
	AccommodationProductInfoServiceV2Request  types.MessageType = types.MessageType(AccommodationProductInfoServiceV2 + ".Request")
	AccommodationProductInfoServiceV2Response types.MessageType = types.MessageType(AccommodationProductInfoServiceV2 + ".Response")
)

var _ rpc.Client = (*AccommodationProductInfoServiceV2Client)(nil)

func NewAccommodationProductInfoServiceV2(grpcCon *grpc.ClientConn) *AccommodationProductInfoServiceV2Client {
	client := accommodationv2grpc.NewAccommodationProductInfoServiceClient(grpcCon)
	return &AccommodationProductInfoServiceV2Client{client: &client}
}

type AccommodationProductInfoServiceV2Client struct {
	client *accommodationv2grpc.AccommodationProductInfoServiceClient
}
