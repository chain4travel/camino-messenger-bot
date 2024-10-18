// Code generated by './scripts/generate_grpc_service_handlers.sh'. DO NOT EDIT.
// template: templates/client.go.tpl

package generated

import (
	"buf.build/gen/go/chain4travel/camino-messenger-protocol/grpc/go/cmp/services/book/v2/bookv2grpc"
	"github.com/chain4travel/camino-messenger-bot/internal/messaging/types"
	"github.com/chain4travel/camino-messenger-bot/internal/rpc"

	"google.golang.org/grpc"
)

const (
	MintServiceV2                           = "cmp.services.book.v2.MintService"
	MintServiceV2Request  types.MessageType = types.MessageType(MintServiceV2 + ".Request")
	MintServiceV2Response types.MessageType = types.MessageType(MintServiceV2 + ".Response")
)

var _ rpc.Client = (*MintServiceV2Client)(nil)

func NewMintServiceV2(grpcCon *grpc.ClientConn) *MintServiceV2Client {
	client := bookv2grpc.NewMintServiceClient(grpcCon)
	return &MintServiceV2Client{client: &client}
}

type MintServiceV2Client struct {
	client *bookv2grpc.MintServiceClient
}
