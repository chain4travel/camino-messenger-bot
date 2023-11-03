package server

import (
	"context"
	"fmt"
	"log"
	"net"

	"camino-messenger-bot/config"
	"camino-messenger-bot/internal/messaging"
	"camino-messenger-bot/internal/proto/pb"
	"go.uber.org/zap"
	"google.golang.org/grpc/metadata"

	"google.golang.org/grpc"
)

var (
	_ Server = (*server)(nil)
)

type Server interface {
	Start()
	Stop()
}
type server struct {
	pb.GreetingServiceServer
	grpcServer *grpc.Server
	cfg        *config.RPCServerConfig
	logger     *zap.SugaredLogger
	processor  messaging.Processor
}

func NewServer(cfg *config.RPCServerConfig, logger *zap.SugaredLogger, opts []grpc.ServerOption, processor messaging.Processor) *server {
	// TODO TLS creds?
	grpcServer := grpc.NewServer(opts...)
	server := &server{grpcServer: grpcServer, cfg: cfg, logger: logger, processor: processor}
	pb.RegisterGreetingServiceServer(grpcServer, server)
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

func (s *server) Greeting(ctx context.Context, req *pb.GreetingServiceRequest) (*pb.GreetingServiceReply, error) {
	md, ok := metadata.FromIncomingContext(ctx)
	if !ok {
		return nil, fmt.Errorf("missing metadata")
	}

	sender := md.Get("sender")
	m := &messaging.Message{
		RequestID: "",
		Type:      messaging.HotelAvailRequest, // TODO remove
		Body:      "",
		Metadata:  messaging.Metadata{},
	}
	m.Metadata.Sender = sender[0]

	response, err := s.processor.ProcessOutbound(ctx, *m)
	return &pb.GreetingServiceReply{
		Message: fmt.Sprintf("Hello, %s", response.RequestID),
	}, err
}
