// Code generated by 'scripts/generate_grpc_service_handlers.sh'. DO NOT EDIT.
// template: templates/client.go.tpl

package generated

import (
	"buf.build/gen/go/chain4travel/camino-messenger-protocol/grpc/go/cmp/services/activity/v1/activityv1grpc"
	"github.com/chain4travel/camino-messenger-bot/internal/messaging/types"
	"github.com/chain4travel/camino-messenger-bot/internal/rpc"

	"google.golang.org/grpc"
)

const (
	ActivitySearchServiceV1                           = "cmp.services.activity.v1.ActivitySearchService"
	ActivitySearchServiceV1Request  types.MessageType = types.MessageType(ActivitySearchServiceV1 + ".Request")
	ActivitySearchServiceV1Response types.MessageType = types.MessageType(ActivitySearchServiceV1 + ".Response")
)

var _ rpc.Client = (*ActivitySearchServiceV1Client)(nil)

func NewActivitySearchServiceV1(grpcCon *grpc.ClientConn) *ActivitySearchServiceV1Client {
	client := activityv1grpc.NewActivitySearchServiceClient(grpcCon)
	return &ActivitySearchServiceV1Client{client: &client}
}

type ActivitySearchServiceV1Client struct {
	client *activityv1grpc.ActivitySearchServiceClient
}
