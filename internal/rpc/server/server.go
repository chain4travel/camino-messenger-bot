package server

import (
	"context"
	"fmt"
	"net"

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
	_ Server = (*server)(nil)
)

type externalRequestProcessor interface {
	processExternalRequest(ctx context.Context, requestType messaging.MessageType, request protoreflect.ProtoMessage) (protoreflect.ProtoMessage, error)
}

type Server interface {
	metadata.Checkpoint
	Start() error
	Stop()
}
type server struct {
	grpcServer      *grpc.Server
	cfg             *config.RPCServerConfig
	logger          *zap.SugaredLogger
	tracer          tracing.Tracer
	processor       messaging.Processor
	serviceRegistry messaging.ServiceRegistry
}

func (*server) Checkpoint() string {
	return "request-gateway"
}

func NewServer(cfg *config.RPCServerConfig, logger *zap.SugaredLogger, tracer tracing.Tracer, processor messaging.Processor, serviceRegistry messaging.ServiceRegistry) Server {
	var opts []grpc.ServerOption
	if cfg.Unencrypted {
		logger.Warn("Running gRPC server without TLS!")
	} else {
		creds, err := utils.LoadTLSCredentials(cfg.ServerCertFile, cfg.ServerKeyFile)
		if err != nil {
			logger.Fatalf("could not load TLS keys: %s", err)
		}
		opts = []grpc.ServerOption{grpc.Creds(creds)}
	}
	server := &server{cfg: cfg, logger: logger, tracer: tracer, processor: processor, serviceRegistry: serviceRegistry}
	server.grpcServer = createGrpcServerAndRegisterServices(server, opts...)
	NewPartnerSrv1(server.grpcServer, server)
	NewPartnerSrv2(server.grpcServer, server)
	return server
}

func createGrpcServerAndRegisterServices(server *server, opts ...grpc.ServerOption) *grpc.Server {
	grpcServer := grpc.NewServer(opts...)
	return grpcServer
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
func (s *server) processInternalRequest(ctx context.Context, requestType messaging.MessageType, request protoreflect.ProtoMessage) (protoreflect.ProtoMessage, error) {
	ctx, span := s.tracer.Start(ctx, "server.processInternalRequest", trace.WithSpanKind(trace.SpanKindServer))
	defer span.End()
	service, registered := s.serviceRegistry.GetService(requestType)
	if !registered {
		return nil, fmt.Errorf("%w: %s", messaging.ErrUnsupportedService, requestType)
	}
	response, _, err := service.Call(ctx, request)
	return response, err
}

func (s *server) processExternalRequest(ctx context.Context, requestType messaging.MessageType, request protoreflect.ProtoMessage) (protoreflect.ProtoMessage, error) {
	ctx, span := s.tracer.Start(ctx, "server.processExternalRequest", trace.WithSpanKind(trace.SpanKindServer))
	defer span.End()
	md, err := s.processMetadata(ctx, s.tracer.TraceIDForSpan(span))
	if err != nil {
		return nil, fmt.Errorf("error processing metadata: %w", err)
	}

	m := &messaging.Message{
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
