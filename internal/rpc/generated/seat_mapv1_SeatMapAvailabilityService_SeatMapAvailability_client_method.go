// Code generated by './scripts/generate_grpc_service_handlers.sh'. DO NOT EDIT.
// template: templates/client_method.go.tpl

package generated

import (
	"context"
	"fmt"

	seat_mapv1 "buf.build/gen/go/chain4travel/camino-messenger-protocol/protocolbuffers/go/cmp/services/seat_map/v1"
	"github.com/chain4travel/camino-messenger-bot/internal/messaging/types"

	"google.golang.org/grpc"
	"google.golang.org/protobuf/reflect/protoreflect"
)

func (s SeatMapAvailabilityServiceV1Client) Call(ctx context.Context, requestIntf protoreflect.ProtoMessage, opts ...grpc.CallOption) (protoreflect.ProtoMessage, types.MessageType, error) {
	request, ok := requestIntf.(*seat_mapv1.SeatMapAvailabilityRequest)
	if !ok {
		return nil, SeatMapAvailabilityServiceV1Response, fmt.Errorf("invalid request type")
	}
	response, err := (*s.client).SeatMapAvailability(ctx, request, opts...)
	return response, SeatMapAvailabilityServiceV1Response, err
}
