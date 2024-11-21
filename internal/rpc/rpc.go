package rpc

import (
	"context"

	"github.com/chain4travel/camino-messenger-bot/internal/messaging/types"
	"google.golang.org/grpc"
	"google.golang.org/protobuf/reflect/protoreflect"
)

type RequestHandler interface {
	HandleRequest(ctx context.Context, requestType types.MessageType, request protoreflect.ProtoMessage) (protoreflect.ProtoMessage, error)
}

type Client interface {
	Call(ctx context.Context, request protoreflect.ProtoMessage, opts ...grpc.CallOption) (protoreflect.ProtoMessage, types.MessageType, error)
}

type ServiceRegistry interface {
	GetService(requestType types.MessageType) (Service, bool)
}

var _ Service = (*service)(nil)

type Service interface {
	Name() string

	Client
}

func NewService(client Client, name string) Service {
	return &service{
		client: client,
		name:   name,
	}
}

type service struct {
	client Client
	name   string
}

func (s *service) Call(ctx context.Context, request protoreflect.ProtoMessage, opts ...grpc.CallOption) (protoreflect.ProtoMessage, types.MessageType, error) {
	return s.client.Call(ctx, request, opts...)
}

func (s *service) Name() string {
	return s.name
}
