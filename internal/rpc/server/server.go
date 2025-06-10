// Copyright (C) 2022-2025, Chain4Travel AG. All rights reserved.
// See the file LICENSE for licensing terms.

package server

import (
	"context"
	"errors"
	"fmt"
	"net"

	"buf.build/gen/go/chain4travel/camino-messenger-protocol/grpc/go/cmp/services/cancellation/v1/cancellationv1grpc"
	"github.com/chain4travel/camino-messenger-bot/v11/config"
	"github.com/chain4travel/camino-messenger-bot/v11/internal/common"
	"github.com/chain4travel/camino-messenger-bot/v11/internal/messaging"
	"github.com/chain4travel/camino-messenger-bot/v11/internal/messaging/types"
	"github.com/chain4travel/camino-messenger-bot/v11/internal/rpc"
	"github.com/chain4travel/camino-messenger-bot/v11/internal/rpc/generated"
	"github.com/chain4travel/camino-messenger-bot/v11/internal/tracing"
	"github.com/chain4travel/camino-messenger-bot/v11/internal/utils/tls"
	"github.com/chain4travel/camino-messenger-bot/v11/pkg/metadata"
	"github.com/chain4travel/camino-messenger-bot/v11/proto/pb/readiness"
	"github.com/grpc-ecosystem/go-grpc-middleware/v2/interceptors"
	"github.com/grpc-ecosystem/go-grpc-middleware/v2/interceptors/selector"

	"go.opentelemetry.io/otel/trace"
	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"
	"google.golang.org/protobuf/reflect/protoreflect"
)

var (
	_ Server                           = (*server)(nil)
	_ rpc.RequestHandler               = (*server)(nil)
	_ readiness.ReadinessServiceServer = (*server)(nil)
)

type Server interface {
	Start() (chan error, error)
	Stop()
}

func NewServer(
	cfg config.RPCServerConfig,
	logger *zap.SugaredLogger,
	responseHeaderHandler common.ResponseHeaderHandler,
	tracer tracing.Tracer,
	processor messaging.MessageProcessor,
	serviceRegistry messaging.ServiceRegistry,
	cancellationV1Service cancellationv1grpc.CancellationServiceServer,
	developerMode bool,
) (Server, error) {
	if !cfg.Enabled {
		return nil, nil
	}

	var opts []grpc.ServerOption
	if cfg.Unencrypted {
		logger.Warn("Running gRPC server without TLS!")
	} else {
		creds, err := tls.LoadTLSCredentials(cfg.ServerCertFile, cfg.ServerKeyFile)
		if err != nil {
			return nil, fmt.Errorf("could not load TLS keys: %w", err)
		}
		opts = []grpc.ServerOption{grpc.Creds(creds)}
	}

	s := &server{
		cfg:                   cfg,
		logger:                logger,
		responseHeaderHandler: responseHeaderHandler,
		tracer:                tracer,
		processor:             processor,
		serviceRegistry:       serviceRegistry,
	}

	opts = append(opts, grpc.UnaryInterceptor(
		selector.UnaryServerInterceptor( // for all cancellationv1grpc methods
			s.errorHandlingInterceptor,
			selector.MatchFunc(func(_ context.Context, callMeta interceptors.CallMeta) bool {
				return cancellationv1grpc.CancellationService_ServiceDesc.ServiceName == callMeta.Service
			}),
		),
	))

	s.grpcServer = grpc.NewServer(opts...)
	generated.RegisterServerServices(s.grpcServer, s)
	cancellationv1grpc.RegisterCancellationServiceServer(s.grpcServer, cancellationV1Service)
	readiness.RegisterReadinessServiceServer(s.grpcServer, s)

	// Register reflection service on gRPC server in developerMode.
	if developerMode {
		reflection.Register(s.grpcServer)
	}
	return s, nil
}

type server struct {
	grpcServer            *grpc.Server
	cfg                   config.RPCServerConfig
	logger                *zap.SugaredLogger
	responseHeaderHandler common.ResponseHeaderHandler
	tracer                tracing.Tracer
	processor             messaging.MessageProcessor
	serviceRegistry       messaging.ServiceRegistry

	readiness.UnimplementedReadinessServiceServer
}

func (*server) checkpoint() string {
	return "request-gateway"
}

func (s *server) Start() (chan error, error) {
	lis, err := net.Listen("tcp", fmt.Sprintf(":%d", s.cfg.Port))
	if err != nil {
		return nil, fmt.Errorf("failed to listen: %w", err)
	}

	errChan := make(chan error)

	go func() {
		defer close(errChan)

		s.logger.Infof("gRPC server listening on %s", lis.Addr().String())

		if err := s.grpcServer.Serve(lis); err != nil && !errors.Is(err, context.Canceled) && !errors.Is(err, grpc.ErrServerStopped) {
			s.logger.Errorf("gRPC server stopped serving with error: %w", err)
			errChan <- err
		}
	}()

	return errChan, nil
}

func (s *server) Stop() {
	s.grpcServer.Stop()
}

func (s *server) HandleMessageRequest(ctx context.Context, requestType types.MessageType, request protoreflect.ProtoMessage) (protoreflect.ProtoMessage, error) {
	ctx, span := s.tracer.Start(ctx, "server.HandleMessageRequest", trace.WithSpanKind(trace.SpanKindServer))
	defer span.End()
	md, err := s.processMetadata(ctx, s.tracer.TraceIDForSpan(span))
	if err != nil {
		return nil, fmt.Errorf("error processing metadata: %w", err)
	}

	response, err := s.processor.SendRequestMessage(ctx, &types.Message{
		Type:     requestType,
		Content:  request,
		Metadata: md,
	})
	if err != nil {
		return nil, fmt.Errorf("error processing outbound request: %w", err)
	}
	response.Metadata.Stamp(fmt.Sprintf("%s-%s", s.checkpoint(), "processed"))

	// TODO @nikos set specific errors according to https://grpc.github.io/grpc/core/md_doc_statuscodes.html ?
	return response.Content, grpc.SendHeader(ctx, response.Metadata.ToGrpcMD())
}

func (s *server) processMetadata(ctx context.Context, id trace.TraceID) (metadata.Metadata, error) {
	md := metadata.Metadata{
		RequestID: id.String(),
	}
	md.Stamp(fmt.Sprintf("%s-%s", s.checkpoint(), "received"))
	err := md.ExtractMetadata(ctx)
	return md, err
}

func (s *server) errorHandlingInterceptor(
	ctx context.Context,
	request any,
	_ *grpc.UnaryServerInfo,
	handler grpc.UnaryHandler,
) (response any, err error) {
	response, err = handler(ctx, request)
	if err != nil {
		responseProtoMessage, ok := response.(protoreflect.ProtoMessage)
		if !ok {
			return response, err
		}
		s.responseHeaderHandler.AddError(responseProtoMessage, err.Error())
	}
	return response, nil
}
