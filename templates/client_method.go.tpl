// Code generated by '{{GENERATOR}}'. DO NOT EDIT.
// template: {{TEMPLATE}}

package generated

import (
	"context"
	"fmt"

	{{TYPE_PACKAGE}} "{{PROTO_INC}}"
	"github.com/chain4travel/camino-messenger-bot/internal/messaging/types"

	"google.golang.org/grpc"
	"google.golang.org/protobuf/reflect/protoreflect"
)

func (s {{SERVICE}}V{{VERSION}}Client) Call(ctx context.Context, requestIntf protoreflect.ProtoMessage, opts ...grpc.CallOption) (protoreflect.ProtoMessage, types.MessageType, error) {
	request, ok := requestIntf.(*{{TYPE_PACKAGE}}.{{REQUEST}})
	if !ok {
		return nil, {{SERVICE}}V{{VERSION}}Response, fmt.Errorf("invalid request type")
	}
	response, err := (*s.client).{{METHOD}}(ctx, request, opts...)
	return response, {{SERVICE}}V{{VERSION}}Response, err
}
