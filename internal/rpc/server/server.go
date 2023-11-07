package server

import (
	"context"
	"fmt"
	"log"
	"net"

	"camino-messenger-bot/config"
	"camino-messenger-bot/internal/messaging"
	"camino-messenger-bot/internal/metadata"
	pb2 "camino-messenger-bot/proto/pb"

	"github.com/google/uuid"
	"go.uber.org/zap"
	"google.golang.org/grpc"
)

var (
	_ Server = (*server)(nil)
)

type Server interface {
	metadata.Checkpoint
	Start()
	Stop()
}
type server struct {
	pb2.GreetingServiceServer
	grpcServer *grpc.Server
	cfg        *config.RPCServerConfig
	logger     *zap.SugaredLogger
	processor  messaging.Processor
}

func (s *server) Checkpoint() string {
	return "request-gateway"
}

func NewServer(cfg *config.RPCServerConfig, logger *zap.SugaredLogger, opts []grpc.ServerOption, processor messaging.Processor) *server {
	// TODO TLS creds?
	grpcServer := grpc.NewServer(opts...)
	server := &server{grpcServer: grpcServer, cfg: cfg, logger: logger, processor: processor}
	pb2.RegisterGreetingServiceServer(grpcServer, server)
	return server
}

func (s *server) Start() {
	lis, err := net.Listen("tcp", fmt.Sprintf("localhost:%d", s.cfg.RPCServerPort))
	if err != nil {
		log.Fatalf("failed to listen: %v", err)
	}
	s.grpcServer.Serve(lis)
}

func (s *server) Stop() {
	s.logger.Info("Stopping gRPC server...")
	s.grpcServer.Stop()
}

func (s *server) Greeting(ctx context.Context, req *pb2.GreetingServiceRequest) (*pb2.GreetingServiceReply, error) {
	requestID, err := uuid.NewRandom()
	if err != nil {
		return nil, err
	}

	md := metadata.Metadata{
		RequestID: requestID.String(),
	}
	md.Stamp(fmt.Sprintf("%s-%s", s.Checkpoint(), "received"))
	err = md.ExtractMetadata(ctx)
	if err != nil {
		return nil, fmt.Errorf("error extracting metadata")
	}

	m := &messaging.Message{
		Type:     messaging.HotelAvailRequest, // TODO remove
		Body:     "",
		Metadata: md,
	}

	response, err := s.processor.ProcessOutbound(ctx, *m)
	md.Stamp(fmt.Sprintf("%s-%s", s.Checkpoint(), "processed"))
	return &pb2.GreetingServiceReply{
		Message: fmt.Sprintf("Hello, %s", response.Metadata.RequestID),
	}, err
}
