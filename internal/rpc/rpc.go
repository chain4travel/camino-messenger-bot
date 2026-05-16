// Copyright (C) 2022-2026, Chain4Travel AG. All rights reserved.
// See the file LICENSE for licensing terms.

package rpc

import (
	"context"
	"errors"

	"github.com/chain4travel/camino-messenger-bot/v13/internal/messaging/message"
	"google.golang.org/grpc"
	"google.golang.org/protobuf/reflect/protoreflect"
)

var (
	ErrNilResponseHeader = errors.New("response header is nil")
	ErrInvalidProto      = errors.New("invalid proto message")
	ErrBlockchain        = errors.New("blockchain error")
	ErrBusinessProcess   = errors.New("business process error")
)

type RequestHandler interface {
	HandleMessageRequest(ctx context.Context, requestType message.Type, request protoreflect.ProtoMessage) (protoreflect.ProtoMessage, error)
}

type Client interface {
	Call(ctx context.Context, request protoreflect.ProtoMessage, opts ...grpc.CallOption) (protoreflect.ProtoMessage, message.Type)
	InvalidProtoErrResponseAndType(errorMessage string) (protoreflect.ProtoMessage, message.Type)
}

type ServiceRegistry interface {
	GetService(requestType message.Type) (Service, bool)
}

var _ Service = (*service)(nil)

type Service interface {
	Name() string

	Client
}

func NewService(client Client, name string) Service {
	return &service{
		Client: client,
		name:   name,
	}
}

type service struct {
	Client
	name string
}

func (s *service) Name() string {
	return s.name
}
