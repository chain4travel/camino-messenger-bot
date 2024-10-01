// Code generated by '{{GENERATOR}}'. DO NOT EDIT.
// template: {{TEMPLATE}}

package clients

import (
	"context"
	"fmt"

	"{{GRPC_INC}}"
	{{TYPE_PACKAGE}} "{{PROTO_INC}}"
	"github.com/chain4travel/camino-messenger-bot/internal/messaging/types"

	"google.golang.org/grpc"
	"google.golang.org/protobuf/reflect/protoreflect"
)

const (
	{{SERVICE}}V{{VERSION}} = {{FQPN}}
	{{SERVICE}}V{{VERSION}}Request  types.MessageType = types.MessageType({{SERVICE}}V{{VERSION}} + ".Request")
	{{SERVICE}}V{{VERSION}}Response types.MessageType = types.MessageType({{SERVICE}}V{{VERSION}} + ".Response")
)

var _ Client = (*{{SERVICE}}Client)(nil)

func New{{SERVICE}}V{{VERSION}}(grpcCon *grpc.ClientConn) *{{SERVICE}}Client {
	client := {{GRPC_PACKAGE}}.New{{SERVICE}}Client(grpcCon)
	return &{{SERVICE}}Client{client: &client}
}

type {{SERVICE}}Client struct {
	client *{{GRPC_PACKAGE}}.{{SERVICE}}Client
}

func (s {{SERVICE}}Client) Call(ctx context.Context, requestIntf protoreflect.ProtoMessage, opts ...grpc.CallOption) (protoreflect.ProtoMessage, types.MessageType, error) {
	request, ok := requestIntf.(*{{TYPE_PACKAGE}}.{{REQUEST}}))
	if !ok {
		return nil, {{SERVICE}}V{{VERSION}}Response, fmt.Errorf("invalid request type")
	}
	response, err := (*s.client).{{METHOD}})(ctx, request, opts...)
	return response, {{SERVICE}}V{{VERSION}}Response, err
}
