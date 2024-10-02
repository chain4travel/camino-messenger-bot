// Code generated by 'scripts/generate_grpc_service_handlers.sh'. DO NOT EDIT.
// template: templates/client.go.tpl

package generated

import (
	"buf.build/gen/go/chain4travel/camino-messenger-protocol/grpc/go/cmp/services/book/v1/bookv1grpc"
	"github.com/chain4travel/camino-messenger-bot/internal/messaging/types"
	"github.com/chain4travel/camino-messenger-bot/internal/rpc"

	"google.golang.org/grpc"
)

const (
	MintServiceV1                           = "cmp.services.book.v1.MintService"
	MintServiceV1Request  types.MessageType = types.MessageType(MintServiceV1 + ".Request")
	MintServiceV1Response types.MessageType = types.MessageType(MintServiceV1 + ".Response")
)

var _ rpc.Client = (*MintServiceV1Client)(nil)

func NewMintServiceV1(grpcCon *grpc.ClientConn) *MintServiceV1Client {
	client := bookv1grpc.NewMintServiceClient(grpcCon)
	return &MintServiceV1Client{client: &client}
}

type MintServiceV1Client struct {
	client *bookv1grpc.MintServiceClient
}
