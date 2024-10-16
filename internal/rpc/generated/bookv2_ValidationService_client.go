// Code generated by 'scripts/generate_grpc_service_handlers.sh'. DO NOT EDIT.
// template: templates/client.go.tpl

package generated

import (
	"buf.build/gen/go/chain4travel/camino-messenger-protocol/grpc/go/cmp/services/book/v2/bookv2grpc"
	"github.com/chain4travel/camino-messenger-bot/internal/messaging/types"
	"github.com/chain4travel/camino-messenger-bot/internal/rpc"

	"google.golang.org/grpc"
)

const (
	ValidationServiceV2                           = "cmp.services.book.v2.ValidationService"
	ValidationServiceV2Request  types.MessageType = types.MessageType(ValidationServiceV2 + ".Request")
	ValidationServiceV2Response types.MessageType = types.MessageType(ValidationServiceV2 + ".Response")
)

var _ rpc.Client = (*ValidationServiceV2Client)(nil)

func NewValidationServiceV2(grpcCon *grpc.ClientConn) *ValidationServiceV2Client {
	client := bookv2grpc.NewValidationServiceClient(grpcCon)
	return &ValidationServiceV2Client{client: &client}
}

type ValidationServiceV2Client struct {
	client *bookv2grpc.ValidationServiceClient
}
