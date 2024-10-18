// Code generated by './scripts/generate_grpc_service_handlers.sh'. DO NOT EDIT.
// template: templates/client.go.tpl

package generated

import (
	"buf.build/gen/go/chain4travel/camino-messenger-protocol/grpc/go/cmp/services/info/v2/infov2grpc"
	"github.com/chain4travel/camino-messenger-bot/internal/messaging/types"
	"github.com/chain4travel/camino-messenger-bot/internal/rpc"

	"google.golang.org/grpc"
)

const (
	CountryEntryRequirementsServiceV2                           = "cmp.services.info.v2.CountryEntryRequirementsService"
	CountryEntryRequirementsServiceV2Request  types.MessageType = types.MessageType(CountryEntryRequirementsServiceV2 + ".Request")
	CountryEntryRequirementsServiceV2Response types.MessageType = types.MessageType(CountryEntryRequirementsServiceV2 + ".Response")
)

var _ rpc.Client = (*CountryEntryRequirementsServiceV2Client)(nil)

func NewCountryEntryRequirementsServiceV2(grpcCon *grpc.ClientConn) *CountryEntryRequirementsServiceV2Client {
	client := infov2grpc.NewCountryEntryRequirementsServiceClient(grpcCon)
	return &CountryEntryRequirementsServiceV2Client{client: &client}
}

type CountryEntryRequirementsServiceV2Client struct {
	client *infov2grpc.CountryEntryRequirementsServiceClient
}
