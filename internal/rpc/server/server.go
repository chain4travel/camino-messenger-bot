package server

import (
	"context"
	"fmt"
	"net"

	"github.com/chain4travel/camino-messenger-bot/internal/messaging/types"
	"github.com/chain4travel/camino-messenger-bot/internal/rpc"
	"github.com/chain4travel/camino-messenger-bot/internal/rpc/generated"
	"github.com/chain4travel/camino-messenger-bot/internal/tracing"

	"github.com/chain4travel/camino-messenger-bot/config"
	"github.com/chain4travel/camino-messenger-bot/internal/messaging"
	"github.com/chain4travel/camino-messenger-bot/internal/metadata"
	utils "github.com/chain4travel/camino-messenger-bot/utils/tls"
	"go.opentelemetry.io/otel/trace"
	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/protobuf/reflect/protoreflect"
)

var (
	_ Server                       = (*server)(nil)
	_ rpc.ExternalRequestProcessor = (*server)(nil)
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
	processor messaging.Processor,
	serviceRegistry messaging.ServiceRegistry,
) (Server, error) {
	if !cfg.Enabled {
		return nil, nil
	}

	var opts []grpc.ServerOption
	if cfg.Unencrypted {
		logger.Warn("Running gRPC server without TLS!")
	} else {
		creds, err := utils.LoadTLSCredentials(cfg.ServerCertFile, cfg.ServerKeyFile)
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
		grpcServer:      grpc.NewServer(opts...),
	}
	generated.RegisterServerServices(server.grpcServer, server)
	return server, nil
}

type server struct {
	grpcServer      *grpc.Server
	cfg             config.RPCServerConfig
	logger          *zap.SugaredLogger
	tracer          tracing.Tracer
	processor       messaging.Processor
	serviceRegistry messaging.ServiceRegistry
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
	s.logger.Info("Stopping gRPC server...")
	s.grpcServer.Stop()
}

func (s *server) ProcessExternalRequest(ctx context.Context, requestType types.MessageType, request protoreflect.ProtoMessage) (protoreflect.ProtoMessage, error) {
	ctx, span := s.tracer.Start(ctx, "server.processExternalRequest", trace.WithSpanKind(trace.SpanKindServer))
	defer span.End()
	md, err := s.processMetadata(ctx, s.tracer.TraceIDForSpan(span))
	if err != nil {
		return nil, fmt.Errorf("error processing metadata: %w", err)
	}

	m := &types.Message{
		Type:     requestType,
		Content:  request,
		Metadata: md,
	}
	response, err := s.processor.ProcessOutbound(ctx, m)
	if err != nil {
		return nil, fmt.Errorf("error processing outbound request: %w", err)
	}
	response.Metadata.Stamp(fmt.Sprintf("%s-%s", s.Checkpoint(), "processed"))
	err = grpc.SendHeader(ctx, response.Metadata.ToGrpcMD())
	return response.Content, err // TODO set specific errors according to https://grpc.github.io/grpc/core/md_doc_statuscodes.html ?
}

func (s *server) processMetadata(ctx context.Context, id trace.TraceID) (metadata.Metadata, error) {
	md := metadata.Metadata{
		RequestID: id.String(),
	}
	md.Stamp(fmt.Sprintf("%s-%s", s.Checkpoint(), "received"))
	err := md.ExtractMetadata(ctx)
	return md, err
}
