package server

import (
	"context"
	"fmt"
	"net"

	"github.com/chain4travel/camino-messenger-bot/internal/tracing"

	"buf.build/gen/go/chain4travel/camino-messenger-protocol/grpc/go/cmp/services/activity/v1alpha/activityv1alphagrpc"
	"buf.build/gen/go/chain4travel/camino-messenger-protocol/grpc/go/cmp/services/book/v1alpha/bookv1alphagrpc"
	bookv1alpha "buf.build/gen/go/chain4travel/camino-messenger-protocol/protocolbuffers/go/cmp/services/book/v1alpha"

	"buf.build/gen/go/chain4travel/camino-messenger-protocol/grpc/go/cmp/services/network/v1alpha/networkv1alphagrpc"
	"buf.build/gen/go/chain4travel/camino-messenger-protocol/grpc/go/cmp/services/partner/v1alpha/partnerv1alphagrpc"
	"buf.build/gen/go/chain4travel/camino-messenger-protocol/grpc/go/cmp/services/ping/v1alpha/pingv1alphagrpc"
	"buf.build/gen/go/chain4travel/camino-messenger-protocol/grpc/go/cmp/services/transport/v1alpha/transportv1alphagrpc"
	activityv1alpha "buf.build/gen/go/chain4travel/camino-messenger-protocol/protocolbuffers/go/cmp/services/activity/v1alpha"
	networkv1alpha "buf.build/gen/go/chain4travel/camino-messenger-protocol/protocolbuffers/go/cmp/services/network/v1alpha"
	partnerv1alpha "buf.build/gen/go/chain4travel/camino-messenger-protocol/protocolbuffers/go/cmp/services/partner/v1alpha"
	pingv1alpha "buf.build/gen/go/chain4travel/camino-messenger-protocol/protocolbuffers/go/cmp/services/ping/v1alpha"
	transportv1alpha "buf.build/gen/go/chain4travel/camino-messenger-protocol/protocolbuffers/go/cmp/services/transport/v1alpha"

	"buf.build/gen/go/chain4travel/camino-messenger-protocol/grpc/go/cmp/services/accommodation/v1alpha/accommodationv1alphagrpc"
	accommodationv1alpha "buf.build/gen/go/chain4travel/camino-messenger-protocol/protocolbuffers/go/cmp/services/accommodation/v1alpha"
	"github.com/chain4travel/camino-messenger-bot/config"
	"github.com/chain4travel/camino-messenger-bot/internal/messaging"
	"github.com/chain4travel/camino-messenger-bot/internal/metadata"
	utils "github.com/chain4travel/camino-messenger-bot/utils/tls"
	"go.opentelemetry.io/otel/trace"
	"go.uber.org/zap"
	"google.golang.org/grpc"
)

var (
	_ Server = (*server)(nil)

	_ accommodationv1alphagrpc.AccommodationProductInfoServiceServer = (*server)(nil)
	_ accommodationv1alphagrpc.AccommodationProductListServiceServer = (*server)(nil)
	_ accommodationv1alphagrpc.AccommodationSearchServiceServer      = (*server)(nil)
	_ activityv1alphagrpc.ActivityProductListServiceServer           = (*server)(nil)
	_ activityv1alphagrpc.ActivitySearchServiceServer                = (*server)(nil)
	_ networkv1alphagrpc.GetNetworkFeeServiceServer                  = (*server)(nil)
	_ partnerv1alphagrpc.GetPartnerConfigurationServiceServer        = (*server)(nil)
	_ bookv1alphagrpc.MintServiceServer                              = (*server)(nil)
	_ bookv1alphagrpc.ValidationServiceServer                        = (*server)(nil)
	_ pingv1alphagrpc.PingServiceServer                              = (*server)(nil)
	_ transportv1alphagrpc.TransportSearchServiceServer              = (*server)(nil)
)

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
	return server
}

func createGrpcServerAndRegisterServices(server *server, opts ...grpc.ServerOption) *grpc.Server {
	grpcServer := grpc.NewServer(opts...)
	activityv1alphagrpc.RegisterActivityProductListServiceServer(grpcServer, server)
	activityv1alphagrpc.RegisterActivitySearchServiceServer(grpcServer, server)
	accommodationv1alphagrpc.RegisterAccommodationProductInfoServiceServer(grpcServer, server)
	accommodationv1alphagrpc.RegisterAccommodationProductListServiceServer(grpcServer, server)
	accommodationv1alphagrpc.RegisterAccommodationSearchServiceServer(grpcServer, server)
	networkv1alphagrpc.RegisterGetNetworkFeeServiceServer(grpcServer, server)
	partnerv1alphagrpc.RegisterGetPartnerConfigurationServiceServer(grpcServer, server)
	bookv1alphagrpc.RegisterMintServiceServer(grpcServer, server)
	bookv1alphagrpc.RegisterValidationServiceServer(grpcServer, server)
	pingv1alphagrpc.RegisterPingServiceServer(grpcServer, server)
	transportv1alphagrpc.RegisterTransportSearchServiceServer(grpcServer, server)
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

func (s *server) AccommodationProductInfo(ctx context.Context, request *accommodationv1alpha.AccommodationProductInfoRequest) (*accommodationv1alpha.AccommodationProductInfoResponse, error) {
	response, err := s.processExternalRequest(ctx, messaging.AccommodationProductInfoRequest, &messaging.RequestContent{AccommodationProductInfoRequest: request})
	return response.AccommodationProductInfoResponse, err
}

func (s *server) AccommodationProductList(ctx context.Context, request *accommodationv1alpha.AccommodationProductListRequest) (*accommodationv1alpha.AccommodationProductListResponse, error) {
	response, err := s.processExternalRequest(ctx, messaging.AccommodationProductListRequest, &messaging.RequestContent{AccommodationProductListRequest: request})
	return response.AccommodationProductListResponse, err
}

func (s *server) AccommodationSearch(ctx context.Context, request *accommodationv1alpha.AccommodationSearchRequest) (*accommodationv1alpha.AccommodationSearchResponse, error) {
	response, err := s.processExternalRequest(ctx, messaging.AccommodationSearchRequest, &messaging.RequestContent{AccommodationSearchRequest: request})
	return response.AccommodationSearchResponse, err // TODO set specific errors according to https://grpc.github.io/grpc/core/md_doc_statuscodes.html ?
}

func (s *server) Ping(ctx context.Context, request *pingv1alpha.PingRequest) (*pingv1alpha.PingResponse, error) {
	response, err := s.processExternalRequest(ctx, messaging.PingRequest, &messaging.RequestContent{PingRequest: request})
	return response.PingResponse, err
}

func (s *server) GetNetworkFee(ctx context.Context, request *networkv1alpha.GetNetworkFeeRequest) (*networkv1alpha.GetNetworkFeeResponse, error) {
	response, err := s.processInternalRequest(ctx, messaging.GetNetworkFeeRequest, &messaging.RequestContent{GetNetworkFeeRequest: request})
	return response.GetNetworkFeeResponse, err
}

func (s *server) GetPartnerConfiguration(ctx context.Context, request *partnerv1alpha.GetPartnerConfigurationRequest) (*partnerv1alpha.GetPartnerConfigurationResponse, error) {
	response, err := s.processInternalRequest(ctx, messaging.GetPartnerConfigurationRequest, &messaging.RequestContent{GetPartnerConfigurationRequest: request})
	return response.GetPartnerConfigurationResponse, err
}

func (s *server) ActivityProductList(ctx context.Context, request *activityv1alpha.ActivityProductListRequest) (*activityv1alpha.ActivityProductListResponse, error) {
	response, err := s.processExternalRequest(ctx, messaging.ActivityProductListRequest, &messaging.RequestContent{ActivityProductListRequest: request})
	return response.ActivityProductListResponse, err
}

func (s *server) ActivitySearch(ctx context.Context, request *activityv1alpha.ActivitySearchRequest) (*activityv1alpha.ActivitySearchResponse, error) {
	response, err := s.processExternalRequest(ctx, messaging.ActivitySearchRequest, &messaging.RequestContent{ActivitySearchRequest: request})
	return response.ActivitySearchResponse, err
}

func (s *server) Mint(ctx context.Context, request *bookv1alpha.MintRequest) (*bookv1alpha.MintResponse, error) {
	response, err := s.processExternalRequest(ctx, messaging.MintRequest, &messaging.RequestContent{MintRequest: request})
	return response.MintResponse, err
}

func (s *server) Validation(ctx context.Context, request *bookv1alpha.ValidationRequest) (*bookv1alpha.ValidationResponse, error) {
	response, err := s.processExternalRequest(ctx, messaging.ValidationRequest, &messaging.RequestContent{ValidationRequest: request})
	return response.ValidationResponse, err
}

func (s *server) TransportSearch(ctx context.Context, request *transportv1alpha.TransportSearchRequest) (*transportv1alpha.TransportSearchResponse, error) {
	response, err := s.processExternalRequest(ctx, messaging.TransportSearchRequest, &messaging.RequestContent{TransportSearchRequest: request})
	return response.TransportSearchResponse, err
}

func (s *server) processInternalRequest(ctx context.Context, requestType messaging.MessageType, request *messaging.RequestContent) (*messaging.ResponseContent, error) {
	ctx, span := s.tracer.Start(ctx, "server.processInternalRequest", trace.WithSpanKind(trace.SpanKindServer))
	defer span.End()
	service, registered := s.serviceRegistry.GetService(requestType)
	if !registered {
		return nil, fmt.Errorf("%w: %s", messaging.ErrUnsupportedRequestType, requestType)
	}
	response, _, err := service.Call(ctx, request)
	return response, err
}

func (s *server) processExternalRequest(ctx context.Context, requestType messaging.MessageType, request *messaging.RequestContent) (*messaging.ResponseContent, error) {
	ctx, span := s.tracer.Start(ctx, "server.processExternalRequest", trace.WithSpanKind(trace.SpanKindServer))
	defer span.End()
	md, err := s.processMetadata(ctx, s.tracer.TraceIDForSpan(span))
	if err != nil {
		return nil, fmt.Errorf("error processing metadata: %w", err)
	}

	m := &messaging.Message{
		Type: requestType,
		Content: messaging.MessageContent{
			RequestContent: *request,
		},
		Metadata: md,
	}
	response, err := s.processor.ProcessOutbound(ctx, m)
	if err != nil {
		return &messaging.ResponseContent{}, fmt.Errorf("error processing outbound request: %w", err)
	}
	response.Metadata.Stamp(fmt.Sprintf("%s-%s", s.Checkpoint(), "processed"))
	err = grpc.SendHeader(ctx, response.Metadata.ToGrpcMD())
	return &response.Content.ResponseContent, err // TODO set specific errors according to https://grpc.github.io/grpc/core/md_doc_statuscodes.html ?
}

func (s *server) processMetadata(ctx context.Context, id trace.TraceID) (metadata.Metadata, error) {
	md := metadata.Metadata{
		RequestID: id.String(),
	}
	md.Stamp(fmt.Sprintf("%s-%s", s.Checkpoint(), "received"))
	err := md.ExtractMetadata(ctx)
	return md, err
}
