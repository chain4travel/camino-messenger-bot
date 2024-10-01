// Code generated by '/home/evdev/Documents/Chain4Travel/Git/camino-messenger-bot/scripts/generate_grpc_service_handlers.sh'. DO NOT EDIT.
// template: templates/client.go.tpl

package generated

import (
	"buf.build/gen/go/chain4travel/camino-messenger-protocol/grpc/go/cmp/services/transport/v1/transportv1grpc"
	"github.com/chain4travel/camino-messenger-bot/internal/messaging/types"
	"github.com/chain4travel/camino-messenger-bot/internal/rpc"

	"google.golang.org/grpc"
)

const (
	TransportSearchServiceV1                           = "cmp.services.transport.v1.TransportSearchService"
	TransportSearchServiceV1Request  types.MessageType = types.MessageType(TransportSearchServiceV1 + ".Request")
	TransportSearchServiceV1Response types.MessageType = types.MessageType(TransportSearchServiceV1 + ".Response")
)

var _ rpc.Client = (*TransportSearchServiceV1Client)(nil)

func NewTransportSearchServiceV1(grpcCon *grpc.ClientConn) *TransportSearchServiceV1Client {
	client := transportv1grpc.NewTransportSearchServiceClient(grpcCon)
	return &TransportSearchServiceV1Client{client: &client}
}

type TransportSearchServiceV1Client struct {
	client *transportv1grpc.TransportSearchServiceClient
}
