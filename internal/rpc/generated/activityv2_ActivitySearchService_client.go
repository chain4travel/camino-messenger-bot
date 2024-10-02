// Code generated by 'scripts/generate_grpc_service_handlers.sh'. DO NOT EDIT.
// template: templates/client.go.tpl

package generated

import (
	"buf.build/gen/go/chain4travel/camino-messenger-protocol/grpc/go/cmp/services/activity/v2/activityv2grpc"
	"github.com/chain4travel/camino-messenger-bot/internal/messaging/types"
	"github.com/chain4travel/camino-messenger-bot/internal/rpc"

	"google.golang.org/grpc"
)

const (
	ActivitySearchServiceV2                           = "cmp.services.activity.v2.ActivitySearchService"
	ActivitySearchServiceV2Request  types.MessageType = types.MessageType(ActivitySearchServiceV2 + ".Request")
	ActivitySearchServiceV2Response types.MessageType = types.MessageType(ActivitySearchServiceV2 + ".Response")
)

var _ rpc.Client = (*ActivitySearchServiceV2Client)(nil)

func NewActivitySearchServiceV2(grpcCon *grpc.ClientConn) *ActivitySearchServiceV2Client {
	client := activityv2grpc.NewActivitySearchServiceClient(grpcCon)
	return &ActivitySearchServiceV2Client{client: &client}
}

type ActivitySearchServiceV2Client struct {
	client *activityv2grpc.ActivitySearchServiceClient
}