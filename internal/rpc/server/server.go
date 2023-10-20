package server

import (
	"camino-messenger-provider/config"
	"camino-messenger-provider/internal/proto/pb"
	"context"
	"fmt"
	"log"
	"net"

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
}

func NewServer(cfg *config.RPCServerConfig, opts []grpc.ServerOption) *server {
	// TODO TLS creds?
	grpcServer := grpc.NewServer(opts...)
	server := &server{grpcServer: grpcServer, cfg: cfg}
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
	s.grpcServer.Stop()
}

func (s *server) Greeting(_ context.Context, req *pb.GreetingServiceRequest) (*pb.GreetingServiceReply, error) {
	return &pb.GreetingServiceReply{
		Message: fmt.Sprintf("Hello, %s", req.Name),
	}, nil
}
