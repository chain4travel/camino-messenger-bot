package clients

import (
	"context"

	"github.com/chain4travel/camino-messenger-bot/internal/messaging/types"
	"google.golang.org/grpc"
	"google.golang.org/protobuf/reflect/protoreflect"
)

type Client interface {
	Call(ctx context.Context, request protoreflect.ProtoMessage, opts ...grpc.CallOption) (protoreflect.ProtoMessage, types.MessageType, error)
}

func ServiceNameToRequestType(serviceName string) types.MessageType {
	return types.MessageType(serviceName + ".Request")
}
