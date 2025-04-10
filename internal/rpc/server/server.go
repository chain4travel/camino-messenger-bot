// Copyright (C) 2022-2025, Chain4Travel AG. All rights reserved.
// See the file LICENSE for licensing terms.

package server

import (
	"context"
	"fmt"
	"net"

	"github.com/chain4travel/camino-messenger-bot/config"
	"github.com/chain4travel/camino-messenger-bot/internal/local"
	"github.com/chain4travel/camino-messenger-bot/internal/messaging"
	"github.com/chain4travel/camino-messenger-bot/internal/messaging/types"
	"github.com/chain4travel/camino-messenger-bot/internal/metadata"
	"github.com/chain4travel/camino-messenger-bot/internal/rpc"
	"github.com/chain4travel/camino-messenger-bot/internal/rpc/generated"
	"github.com/chain4travel/camino-messenger-bot/internal/tracing"
	"github.com/chain4travel/camino-messenger-bot/internal/utils/tls"
	"github.com/chain4travel/camino-messenger-bot/proto/pb/readiness"

	"go.opentelemetry.io/otel/trace"
	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"
	"google.golang.org/protobuf/reflect/protoreflect"
	emptypb "google.golang.org/protobuf/types/known/emptypb"
)

var (
	_ Server                           = (*server)(nil)
	_ rpc.RequestHandler               = (*server)(nil)
	_ readiness.ReadinessServiceServer = (*server)(nil)
)

type Server interface {
	metadata.Checkpoint
	Start() error
	Stop()
}

func NewServer(
	cfg config.RPCServerConfig,
	logger *zap.SugaredLogger,
	tracer tracing.Tracer,
	processor messaging.MessageProcessor,
	serviceRegistry messaging.ServiceRegistry,
	localService local.Service,
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
	server := &server{
		cfg:             cfg,
		logger:          logger,
		tracer:          tracer,
		processor:       processor,
		serviceRegistry: serviceRegistry,
		localService:    localService,
		grpcServer:      grpc.NewServer(opts...),
	}
	generated.RegisterServerServices(server.grpcServer, server)
	readiness.RegisterReadinessServiceServer(server.grpcServer, server)

	// Register reflection service on gRPC server in developerMode.
	if developerMode {
		reflection.Register(server.grpcServer)
	}
	return server, nil
}

type server struct {
	grpcServer      *grpc.Server
	cfg             config.RPCServerConfig
	logger          *zap.SugaredLogger
	tracer          tracing.Tracer
	processor       messaging.MessageProcessor
	serviceRegistry messaging.ServiceRegistry
	localService    local.Service

	readiness.UnimplementedReadinessServiceServer
}

func (*server) Checkpoint() string {
	return "request-gateway"
}

func (s *server) Start() error {
	lis, err := net.Listen("tcp", fmt.Sprintf(":%d", s.cfg.Port))
	if err != nil {
		return fmt.Errorf("failed to listen: %w", err)
	}
	return s.grpcServer.Serve(lis)
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
	response.Metadata.Stamp(fmt.Sprintf("%s-%s", s.Checkpoint(), "processed"))

	// TODO set specific errors according to https://grpc.github.io/grpc/core/md_doc_statuscodes.html ?
	return response.Content, grpc.SendHeader(ctx, response.Metadata.ToGrpcMD())
}

func (s *server) HandleLocalRequest(ctx context.Context, request protoreflect.ProtoMessage) (protoreflect.ProtoMessage, error) {
	ctx, span := s.tracer.Start(ctx, "server.HandleMessageRequest", trace.WithSpanKind(trace.SpanKindServer))
	defer span.End()
	md, err := s.processMetadata(ctx, s.tracer.TraceIDForSpan(span))
	if err != nil {
		return nil, fmt.Errorf("error processing metadata: %w", err)
	}

	response, err := s.localService.HandleLocalRequest(ctx, &md, request)
	if err != nil {
		return nil, fmt.Errorf("error processing request: %w", err)
	}

	md.Stamp(fmt.Sprintf("%s-%s", s.Checkpoint(), "processed"))

	// TODO set specific errors according to https://grpc.github.io/grpc/core/md_doc_statuscodes.html ?
	return response, grpc.SendHeader(ctx, md.ToGrpcMD())
}

func (s *server) processMetadata(ctx context.Context, id trace.TraceID) (metadata.Metadata, error) {
	md := metadata.Metadata{
		RequestID: id.String(),
	}
	md.Stamp(fmt.Sprintf("%s-%s", s.Checkpoint(), "received"))
	err := md.ExtractMetadata(ctx)
	return md, err
}

const StatusReady = "ready"

func (s *server) Readiness(context.Context, *emptypb.Empty) (*readiness.ReadinessResponse, error) {
	return &readiness.ReadinessResponse{Status: StatusReady}, nil
}
