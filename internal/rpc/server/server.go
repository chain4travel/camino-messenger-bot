package server

import (
	"context"
	"fmt"
	"log"
	"net"

	"github.com/chain4travel/camino-messenger-bot/config"
	"github.com/chain4travel/camino-messenger-bot/internal/messaging"
	"github.com/chain4travel/camino-messenger-bot/internal/metadata"
	"github.com/chain4travel/camino-messenger-bot/proto/pb/messages"
	utils "github.com/chain4travel/camino-messenger-bot/utils/tls"
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
	messages.FlightSearchServiceServer
	grpcServer *grpc.Server
	cfg        *config.RPCServerConfig
	logger     *zap.SugaredLogger
	processor  messaging.Processor
}

func (s *server) Checkpoint() string {
	return "request-gateway"
}

func NewServer(cfg *config.RPCServerConfig, logger *zap.SugaredLogger, processor messaging.Processor) *server {
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
	grpcServer := grpc.NewServer(opts...)
	server := &server{grpcServer: grpcServer, cfg: cfg, logger: logger, processor: processor}
	messages.RegisterFlightSearchServiceServer(grpcServer, server)
	return server
}

func (s *server) Start() {
	lis, err := net.Listen("tcp", fmt.Sprintf("localhost:%d", s.cfg.Port))
	if err != nil {
		log.Fatalf("failed to listen: %v", err)
	}
	s.grpcServer.Serve(lis)
}

func (s *server) Stop() {
	s.logger.Info("Stopping gRPC server...")
	s.grpcServer.Stop()
}

func (s *server) Search(ctx context.Context, request *messages.FlightSearchRequest) (*messages.FlightSearchResponse, error) {
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
		Type: messaging.FlightSearchRequest,
		Content: messaging.MessageContent{
			RequestContent: messaging.RequestContent{
				FlightSearchRequest: *request,
			},
		},
		Metadata: md,
	}
	response, err := s.processor.ProcessOutbound(ctx, *m)
	response.Metadata.Stamp(fmt.Sprintf("%s-%s", s.Checkpoint(), "processed"))
	grpc.SendHeader(ctx, response.Metadata.ToGrpcMD())
	return &response.Content.ResponseContent.FlightSearchResponse, err //TODO set specific errors according to https://grpc.github.io/grpc/core/md_doc_statuscodes.html ?
}
