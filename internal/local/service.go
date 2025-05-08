// Copyright (C) 2022-2025, Chain4Travel AG. All rights reserved.
// See the file LICENSE for licensing terms.

package local

import (
	"context"
	"errors"
	"fmt"

	"github.com/chain4travel/camino-messenger-bot/v11/pkg/metadata"
	"go.uber.org/zap"
	"google.golang.org/protobuf/reflect/protoreflect"
)

var _ Service = (*service)(nil)

func NewService(
	logger *zap.SugaredLogger,
) Service {
	return &service{
		logger: logger,
	}
}

type Service interface {
	HandleLocalRequest(
		ctx context.Context,
		metadata *metadata.Metadata,
		request protoreflect.ProtoMessage,
	) (protoreflect.ProtoMessage, error)
}

type service struct {
	logger *zap.SugaredLogger
}

func (s *service) HandleLocalRequest(
	_ context.Context,
	metadata *metadata.Metadata,
	_ protoreflect.ProtoMessage,
) (protoreflect.ProtoMessage, error) {
	metadata.Stamp(fmt.Sprintf("%s-%s", s.checkpoint(), "request"))
	return nil, errors.New("not implemented")
}

func (*service) checkpoint() string {
	return "service"
}
