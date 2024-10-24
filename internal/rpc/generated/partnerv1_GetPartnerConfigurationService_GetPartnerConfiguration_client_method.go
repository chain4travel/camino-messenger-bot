// Code generated by 'scripts/generate_grpc_service_handlers.sh'. DO NOT EDIT.
// template: templates/client_method.go.tpl

package generated

import (
	"context"
	"fmt"

	partnerv1 "buf.build/gen/go/chain4travel/camino-messenger-protocol/protocolbuffers/go/cmp/services/partner/v1"
	"github.com/chain4travel/camino-messenger-bot/internal/messaging/types"

	"google.golang.org/grpc"
	"google.golang.org/protobuf/reflect/protoreflect"
)

func (s GetPartnerConfigurationServiceV1Client) Call(ctx context.Context, requestIntf protoreflect.ProtoMessage, opts ...grpc.CallOption) (protoreflect.ProtoMessage, types.MessageType, error) {
	request, ok := requestIntf.(*partnerv1.GetPartnerConfigurationRequest)
	if !ok {
		return nil, GetPartnerConfigurationServiceV1Response, fmt.Errorf("invalid request type")
	}
	response, err := (*s.client).GetPartnerConfiguration(ctx, request, opts...)
	return response, GetPartnerConfigurationServiceV1Response, err
}
