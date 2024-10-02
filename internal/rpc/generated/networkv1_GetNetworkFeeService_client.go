// Code generated by 'scripts/generate_grpc_service_handlers.sh'. DO NOT EDIT.
// template: templates/client.go.tpl

package generated

import (
	"buf.build/gen/go/chain4travel/camino-messenger-protocol/grpc/go/cmp/services/network/v1/networkv1grpc"
	"github.com/chain4travel/camino-messenger-bot/internal/messaging/types"
	"github.com/chain4travel/camino-messenger-bot/internal/rpc"

	"google.golang.org/grpc"
)

const (
	GetNetworkFeeServiceV1                           = "cmp.services.network.v1.GetNetworkFeeService"
	GetNetworkFeeServiceV1Request  types.MessageType = types.MessageType(GetNetworkFeeServiceV1 + ".Request")
	GetNetworkFeeServiceV1Response types.MessageType = types.MessageType(GetNetworkFeeServiceV1 + ".Response")
)

var _ rpc.Client = (*GetNetworkFeeServiceV1Client)(nil)

func NewGetNetworkFeeServiceV1(grpcCon *grpc.ClientConn) *GetNetworkFeeServiceV1Client {
	client := networkv1grpc.NewGetNetworkFeeServiceClient(grpcCon)
	return &GetNetworkFeeServiceV1Client{client: &client}
}

type GetNetworkFeeServiceV1Client struct {
	client *networkv1grpc.GetNetworkFeeServiceClient
}