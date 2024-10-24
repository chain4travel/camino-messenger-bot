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
	ValidationServiceV1                           = "cmp.services.book.v1.ValidationService"
	ValidationServiceV1Request  types.MessageType = types.MessageType(ValidationServiceV1 + ".Request")
	ValidationServiceV1Response types.MessageType = types.MessageType(ValidationServiceV1 + ".Response")
)

var _ rpc.Client = (*ValidationServiceV1Client)(nil)

func NewValidationServiceV1(grpcCon *grpc.ClientConn) *ValidationServiceV1Client {
	client := bookv1grpc.NewValidationServiceClient(grpcCon)
	return &ValidationServiceV1Client{client: &client}
}

type ValidationServiceV1Client struct {
	client *bookv1grpc.ValidationServiceClient
}
