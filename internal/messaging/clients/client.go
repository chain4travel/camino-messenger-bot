package clients

import (
	"context"

	"github.com/chain4travel/camino-messenger-bot/internal/messaging/messages"
	"google.golang.org/grpc"
	"google.golang.org/protobuf/reflect/protoreflect"
)

type Client interface {
	Call(ctx context.Context, request protoreflect.ProtoMessage, opts ...grpc.CallOption) (protoreflect.ProtoMessage, messages.MessageType, error)
}
