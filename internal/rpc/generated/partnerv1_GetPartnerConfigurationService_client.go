// Code generated by './scripts/generate_grpc_service_handlers.sh'. DO NOT EDIT.
// template: templates/client.go.tpl

package generated

import (
	"buf.build/gen/go/chain4travel/camino-messenger-protocol/grpc/go/cmp/services/partner/v1/partnerv1grpc"
	"github.com/chain4travel/camino-messenger-bot/internal/messaging/types"
	"github.com/chain4travel/camino-messenger-bot/internal/rpc"

	"google.golang.org/grpc"
)

const (
	GetPartnerConfigurationServiceV1                           = "cmp.services.partner.v1.GetPartnerConfigurationService"
	GetPartnerConfigurationServiceV1Request  types.MessageType = types.MessageType(GetPartnerConfigurationServiceV1 + ".Request")
	GetPartnerConfigurationServiceV1Response types.MessageType = types.MessageType(GetPartnerConfigurationServiceV1 + ".Response")
)

var _ rpc.Client = (*GetPartnerConfigurationServiceV1Client)(nil)

func NewGetPartnerConfigurationServiceV1(grpcCon *grpc.ClientConn) *GetPartnerConfigurationServiceV1Client {
	client := partnerv1grpc.NewGetPartnerConfigurationServiceClient(grpcCon)
	return &GetPartnerConfigurationServiceV1Client{client: &client}
}

type GetPartnerConfigurationServiceV1Client struct {
	client *partnerv1grpc.GetPartnerConfigurationServiceClient
}
