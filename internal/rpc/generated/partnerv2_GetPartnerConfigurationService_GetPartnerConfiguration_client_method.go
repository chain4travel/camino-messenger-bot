// Code generated by 'scripts/generate_grpc_service_handlers.sh'. DO NOT EDIT.
// template: templates/client_method.go.tpl

package generated

import (
	"context"
	"fmt"

	partnerv2 "buf.build/gen/go/chain4travel/camino-messenger-protocol/protocolbuffers/go/cmp/services/partner/v2"
	"github.com/chain4travel/camino-messenger-bot/internal/messaging/types"

	"google.golang.org/grpc"
	"google.golang.org/protobuf/reflect/protoreflect"
)

func (s GetPartnerConfigurationServiceV2Client) Call(ctx context.Context, requestIntf protoreflect.ProtoMessage, opts ...grpc.CallOption) (protoreflect.ProtoMessage, types.MessageType, error) {
	request, ok := requestIntf.(*partnerv2.GetPartnerConfigurationRequest)
	if !ok {
		return nil, GetPartnerConfigurationServiceV2Response, fmt.Errorf("invalid request type")
	}
	response, err := (*s.client).GetPartnerConfiguration(ctx, request, opts...)
	return response, GetPartnerConfigurationServiceV2Response, err
}
