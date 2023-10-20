package server

import (
	"camino-messenger-provider/internal/proto/pb"
	"context"
	"flag"
	"fmt"
	"log"
	"net"

	"google.golang.org/grpc"
)

var (
	_          Server = (*server)(nil)
	tls               = flag.Bool("tls", false, "Connection uses TLS if true, else plain TCP")
	certFile          = flag.String("cert_file", "", "The TLS cert file")
	keyFile           = flag.String("key_file", "", "The TLS key file")
	jsonDBFile        = flag.String("json_db_file", "", "A json file containing a list of features")
	port              = flag.Int("port", 50051, "The server port")
)

type Server interface {
	Start()
	Stop()
}
type server struct {
	pb.GreetingServiceServer
	grpcServer *grpc.Server
}

func NewServer(opts []grpc.ServerOption) *server {
	// TODO TLS creds?
	grpcServer := grpc.NewServer(opts...)
	server := &server{grpcServer: grpcServer}
	pb.RegisterGreetingServiceServer(grpcServer, server)
	return server
}

func (s *server) Start() {
	lis, err := net.Listen("tcp", fmt.Sprintf("localhost:%d", *port))
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
