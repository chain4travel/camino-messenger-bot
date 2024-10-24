// Code generated by 'scripts/generate_grpc_service_handlers.sh'. DO NOT EDIT.
// template: templates/client_method.go.tpl

package generated

import (
	"context"
	"fmt"

	accommodationv2 "buf.build/gen/go/chain4travel/camino-messenger-protocol/protocolbuffers/go/cmp/services/accommodation/v2"
	"github.com/chain4travel/camino-messenger-bot/internal/messaging/types"

	"google.golang.org/grpc"
	"google.golang.org/protobuf/reflect/protoreflect"
)

func (s AccommodationProductInfoServiceV2Client) Call(ctx context.Context, requestIntf protoreflect.ProtoMessage, opts ...grpc.CallOption) (protoreflect.ProtoMessage, types.MessageType, error) {
	request, ok := requestIntf.(*accommodationv2.AccommodationProductInfoRequest)
	if !ok {
		return nil, AccommodationProductInfoServiceV2Response, fmt.Errorf("invalid request type")
	}
	response, err := (*s.client).AccommodationProductInfo(ctx, request, opts...)
	return response, AccommodationProductInfoServiceV2Response, err
}
