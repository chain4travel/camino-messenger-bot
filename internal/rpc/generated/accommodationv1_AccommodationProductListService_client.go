// Code generated by 'scripts/generate_grpc_service_handlers.sh'. DO NOT EDIT.
// template: templates/client.go.tpl

package generated

import (
	"buf.build/gen/go/chain4travel/camino-messenger-protocol/grpc/go/cmp/services/accommodation/v1/accommodationv1grpc"
	"github.com/chain4travel/camino-messenger-bot/internal/messaging/types"
	"github.com/chain4travel/camino-messenger-bot/internal/rpc"

	"google.golang.org/grpc"
)

const (
	AccommodationProductListServiceV1                           = "cmp.services.accommodation.v1.AccommodationProductListService"
	AccommodationProductListServiceV1Request  types.MessageType = types.MessageType(AccommodationProductListServiceV1 + ".Request")
	AccommodationProductListServiceV1Response types.MessageType = types.MessageType(AccommodationProductListServiceV1 + ".Response")
)

var _ rpc.Client = (*AccommodationProductListServiceV1Client)(nil)

func NewAccommodationProductListServiceV1(grpcCon *grpc.ClientConn) *AccommodationProductListServiceV1Client {
	client := accommodationv1grpc.NewAccommodationProductListServiceClient(grpcCon)
	return &AccommodationProductListServiceV1Client{client: &client}
}

type AccommodationProductListServiceV1Client struct {
	client *accommodationv1grpc.AccommodationProductListServiceClient
}
