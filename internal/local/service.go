package local

import (
	"context"
	"errors"
	"fmt"

	"github.com/chain4travel/camino-messenger-bot/internal/metadata"
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
	ctx context.Context,
	metadata *metadata.Metadata,
	request protoreflect.ProtoMessage,
) (protoreflect.ProtoMessage, error) {
	metadata.Stamp(fmt.Sprintf("%s-%s", s.Checkpoint(), "request"))
	return nil, errors.New("not implemented")
}

func (*service) Checkpoint() string {
	return "service"
}
