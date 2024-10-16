// Code generated by 'scripts/generate_grpc_service_handlers.sh'. DO NOT EDIT.
// template: templates/client.go.tpl

package generated

import (
	"buf.build/gen/go/chain4travel/camino-messenger-protocol/grpc/go/cmp/services/transport/v2/transportv2grpc"
	"github.com/chain4travel/camino-messenger-bot/internal/messaging/types"
	"github.com/chain4travel/camino-messenger-bot/internal/rpc"

	"google.golang.org/grpc"
)

const (
	TransportSearchServiceV2                           = "cmp.services.transport.v2.TransportSearchService"
	TransportSearchServiceV2Request  types.MessageType = types.MessageType(TransportSearchServiceV2 + ".Request")
	TransportSearchServiceV2Response types.MessageType = types.MessageType(TransportSearchServiceV2 + ".Response")
)

var _ rpc.Client = (*TransportSearchServiceV2Client)(nil)

func NewTransportSearchServiceV2(grpcCon *grpc.ClientConn) *TransportSearchServiceV2Client {
	client := transportv2grpc.NewTransportSearchServiceClient(grpcCon)
	return &TransportSearchServiceV2Client{client: &client}
}

type TransportSearchServiceV2Client struct {
	client *transportv2grpc.TransportSearchServiceClient
}
