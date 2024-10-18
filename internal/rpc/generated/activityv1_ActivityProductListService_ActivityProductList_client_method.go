// Code generated by './scripts/generate_grpc_service_handlers.sh'. DO NOT EDIT.
// template: templates/client_method.go.tpl

package generated

import (
	"context"
	"fmt"

	activityv1 "buf.build/gen/go/chain4travel/camino-messenger-protocol/protocolbuffers/go/cmp/services/activity/v1"
	"github.com/chain4travel/camino-messenger-bot/internal/messaging/types"

	"google.golang.org/grpc"
	"google.golang.org/protobuf/reflect/protoreflect"
)

func (s ActivityProductListServiceV1Client) Call(ctx context.Context, requestIntf protoreflect.ProtoMessage, opts ...grpc.CallOption) (protoreflect.ProtoMessage, types.MessageType, error) {
	request, ok := requestIntf.(*activityv1.ActivityProductListRequest)
	if !ok {
		return nil, ActivityProductListServiceV1Response, fmt.Errorf("invalid request type")
	}
	response, err := (*s.client).ActivityProductList(ctx, request, opts...)
	return response, ActivityProductListServiceV1Response, err
}
