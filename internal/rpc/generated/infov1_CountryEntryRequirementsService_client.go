// Code generated by 'scripts/generate_grpc_service_handlers.sh'. DO NOT EDIT.
// template: templates/client.go.tpl

package generated

import (
	"buf.build/gen/go/chain4travel/camino-messenger-protocol/grpc/go/cmp/services/info/v1/infov1grpc"
	"github.com/chain4travel/camino-messenger-bot/internal/messaging/types"
	"github.com/chain4travel/camino-messenger-bot/internal/rpc"

	"google.golang.org/grpc"
)

const (
	CountryEntryRequirementsServiceV1                           = "cmp.services.info.v1.CountryEntryRequirementsService"
	CountryEntryRequirementsServiceV1Request  types.MessageType = types.MessageType(CountryEntryRequirementsServiceV1 + ".Request")
	CountryEntryRequirementsServiceV1Response types.MessageType = types.MessageType(CountryEntryRequirementsServiceV1 + ".Response")
)

var _ rpc.Client = (*CountryEntryRequirementsServiceV1Client)(nil)

func NewCountryEntryRequirementsServiceV1(grpcCon *grpc.ClientConn) *CountryEntryRequirementsServiceV1Client {
	client := infov1grpc.NewCountryEntryRequirementsServiceClient(grpcCon)
	return &CountryEntryRequirementsServiceV1Client{client: &client}
}

type CountryEntryRequirementsServiceV1Client struct {
	client *infov1grpc.CountryEntryRequirementsServiceClient
}
