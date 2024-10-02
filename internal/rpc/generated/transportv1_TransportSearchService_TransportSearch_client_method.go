// Code generated by 'scripts/generate_grpc_service_handlers.sh'. DO NOT EDIT.
// template: templates/client_method.go.tpl

package generated

import (
	"context"
	"fmt"

	transportv1 "buf.build/gen/go/chain4travel/camino-messenger-protocol/protocolbuffers/go/cmp/services/transport/v1"
	"github.com/chain4travel/camino-messenger-bot/internal/messaging/types"

	"google.golang.org/grpc"
	"google.golang.org/protobuf/reflect/protoreflect"
)

func (s TransportSearchServiceV1Client) Call(ctx context.Context, requestIntf protoreflect.ProtoMessage, opts ...grpc.CallOption) (protoreflect.ProtoMessage, types.MessageType, error) {
	request, ok := requestIntf.(*transportv1.TransportSearchRequest)
	if !ok {
		return nil, TransportSearchServiceV1Response, fmt.Errorf("invalid request type")
	}
	response, err := (*s.client).TransportSearch(ctx, request, opts...)
	return response, TransportSearchServiceV1Response, err
}
