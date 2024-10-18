// Code generated by './scripts/generate_grpc_service_handlers.sh'. DO NOT EDIT.
// template: templates/client_method.go.tpl

package generated

import (
	"context"
	"fmt"

	accommodationv1 "buf.build/gen/go/chain4travel/camino-messenger-protocol/protocolbuffers/go/cmp/services/accommodation/v1"
	"github.com/chain4travel/camino-messenger-bot/internal/messaging/types"

	"google.golang.org/grpc"
	"google.golang.org/protobuf/reflect/protoreflect"
)

func (s AccommodationSearchServiceV1Client) Call(ctx context.Context, requestIntf protoreflect.ProtoMessage, opts ...grpc.CallOption) (protoreflect.ProtoMessage, types.MessageType, error) {
	request, ok := requestIntf.(*accommodationv1.AccommodationSearchRequest)
	if !ok {
		return nil, AccommodationSearchServiceV1Response, fmt.Errorf("invalid request type")
	}
	response, err := (*s.client).AccommodationSearch(ctx, request, opts...)
	return response, AccommodationSearchServiceV1Response, err
}
