package server

import (
	"buf.build/gen/go/chain4travel/camino-messenger-protocol/grpc/go/cmp/services/activity/v1alpha1/activityv1alpha1grpc"
	"buf.build/gen/go/chain4travel/camino-messenger-protocol/grpc/go/cmp/services/network/v1alpha1/networkv1alpha1grpc"
	"buf.build/gen/go/chain4travel/camino-messenger-protocol/grpc/go/cmp/services/partner/v1alpha1/partnerv1alpha1grpc"
	"buf.build/gen/go/chain4travel/camino-messenger-protocol/grpc/go/cmp/services/ping/v1alpha1/pingv1alpha1grpc"
	"buf.build/gen/go/chain4travel/camino-messenger-protocol/grpc/go/cmp/services/transport/v1alpha1/transportv1alpha1grpc"
	activityv1alpha1 "buf.build/gen/go/chain4travel/camino-messenger-protocol/protocolbuffers/go/cmp/services/activity/v1alpha1"
	networkv1alpha1 "buf.build/gen/go/chain4travel/camino-messenger-protocol/protocolbuffers/go/cmp/services/network/v1alpha1"
	partnerv1alpha1 "buf.build/gen/go/chain4travel/camino-messenger-protocol/protocolbuffers/go/cmp/services/partner/v1alpha1"
	pingv1alpha1 "buf.build/gen/go/chain4travel/camino-messenger-protocol/protocolbuffers/go/cmp/services/ping/v1alpha1"
	transportv1alpha1 "buf.build/gen/go/chain4travel/camino-messenger-protocol/protocolbuffers/go/cmp/services/transport/v1alpha1"
	"context"
	"fmt"
	"log"
	"net"

	"buf.build/gen/go/chain4travel/camino-messenger-protocol/grpc/go/cmp/services/accommodation/v1alpha1/accommodationv1alpha1grpc"
	"buf.build/gen/go/chain4travel/camino-messenger-protocol/protocolbuffers/go/cmp/services/accommodation/v1alpha1"
	"github.com/chain4travel/camino-messenger-bot/config"
	"github.com/chain4travel/camino-messenger-bot/internal/messaging"
	"github.com/chain4travel/camino-messenger-bot/internal/metadata"
	utils "github.com/chain4travel/camino-messenger-bot/utils/tls"
	"github.com/google/uuid"
	"go.uber.org/zap"
	"google.golang.org/grpc"
)

var (
	_ Server = (*server)(nil)

	_ accommodationv1alpha1grpc.AccommodationSearchServiceServer = (*server)(nil)
	_ activityv1alpha1grpc.ActivitySearchServiceServer           = (*server)(nil)
	_ networkv1alpha1grpc.GetNetworkFeeServiceServer             = (*server)(nil)
	_ partnerv1alpha1grpc.GetPartnerConfigurationServiceServer   = (*server)(nil)
	_ pingv1alpha1grpc.PingServiceServer                         = (*server)(nil)
	_ transportv1alpha1grpc.TransportSearchServiceServer         = (*server)(nil)
)

type Server interface {
	metadata.Checkpoint
	Start()
	Stop()
}
type server struct {
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
	server := &server{cfg: cfg, logger: logger, processor: processor}
	server.grpcServer = createGrpcServerAndRegisterServices(server, opts...)
	return server
}

func createGrpcServerAndRegisterServices(server *server, opts ...grpc.ServerOption) *grpc.Server {
	grpcServer := grpc.NewServer(opts...)
	activityv1alpha1grpc.RegisterActivitySearchServiceServer(grpcServer, server)
	accommodationv1alpha1grpc.RegisterAccommodationSearchServiceServer(grpcServer, server)
	networkv1alpha1grpc.RegisterGetNetworkFeeServiceServer(grpcServer, server)
	partnerv1alpha1grpc.RegisterGetPartnerConfigurationServiceServer(grpcServer, server)
	pingv1alpha1grpc.RegisterPingServiceServer(grpcServer, server)
	transportv1alpha1grpc.RegisterTransportSearchServiceServer(grpcServer, server)
	return grpcServer
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

func (s *server) AccommodationSearch(ctx context.Context, request *accommodationv1alpha1.AccommodationSearchRequest) (*accommodationv1alpha1.AccommodationSearchResponse, error) {
	response, err := s.processRequest(ctx, messaging.AccommodationSearchRequest, &messaging.RequestContent{AccommodationSearchRequest: *request})
	return &response.AccommodationSearchResponse, err //TODO set specific errors according to https://grpc.github.io/grpc/core/md_doc_statuscodes.html ?
}

func (s *server) Ping(ctx context.Context, request *pingv1alpha1.PingRequest) (*pingv1alpha1.PingResponse, error) {
	response, err := s.processRequest(ctx, messaging.PingRequest, &messaging.RequestContent{PingRequest: *request})
	return &response.PingResponse, err
}

func (s *server) GetNetworkFee(ctx context.Context, request *networkv1alpha1.GetNetworkFeeRequest) (*networkv1alpha1.GetNetworkFeeResponse, error) {
	response, err := s.processRequest(ctx, messaging.GetNetworkFeeRequest, &messaging.RequestContent{GetNetworkFeeRequest: *request})
	return &response.GetNetworkFeeResponse, err
}

func (s *server) GetPartnerConfiguration(ctx context.Context, request *partnerv1alpha1.GetPartnerConfigurationRequest) (*partnerv1alpha1.GetPartnerConfigurationResponse, error) {
	response, err := s.processRequest(ctx, messaging.GetPartnerConfigurationRequest, &messaging.RequestContent{GetPartnerConfigurationRequest: *request})
	return &response.GetPartnerConfigurationResponse, err
}

func (s *server) ActivitySearch(ctx context.Context, request *activityv1alpha1.ActivitySearchRequest) (*activityv1alpha1.ActivitySearchResponse, error) {
	response, err := s.processRequest(ctx, messaging.ActivitySearchRequest, &messaging.RequestContent{ActivitySearchRequest: *request})
	return &response.ActivitySearchResponse, err
}

func (s *server) TransportSearch(ctx context.Context, request *transportv1alpha1.TransportSearchRequest) (*transportv1alpha1.TransportSearchResponse, error) {
	response, err := s.processRequest(ctx, messaging.TransportSearchRequest, &messaging.RequestContent{TransportSearchRequest: *request})
	return &response.TransportSearchResponse, err
}

func (s *server) processRequest(ctx context.Context, requestType messaging.MessageType, request *messaging.RequestContent) (*messaging.ResponseContent, error) {
	err, md := s.processMetadata(ctx)
	if err != nil {
		return nil, fmt.Errorf("error processing metadata: %v", err)
	}

	m := &messaging.Message{
		Type: requestType,
		Content: messaging.MessageContent{
			RequestContent: *request,
		},
		Metadata: md,
	}
	response, err := s.processor.ProcessOutbound(ctx, *m)
	response.Metadata.Stamp(fmt.Sprintf("%s-%s", s.Checkpoint(), "processed"))
	grpc.SendHeader(ctx, response.Metadata.ToGrpcMD())
	return &response.Content.ResponseContent, err //TODO set specific errors according to https://grpc.github.io/grpc/core/md_doc_statuscodes.html ?
}

func (s *server) processMetadata(ctx context.Context) (error, metadata.Metadata) {
	requestID, err := uuid.NewRandom()
	if err != nil {
		return nil, metadata.Metadata{}
	}

	md := metadata.Metadata{
		RequestID: requestID.String(),
	}
	md.Stamp(fmt.Sprintf("%s-%s", s.Checkpoint(), "received"))
	err = md.ExtractMetadata(ctx)
	return err, md
}
