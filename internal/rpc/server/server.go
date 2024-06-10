package server

import (
	"context"
	"fmt"
	"net"

	"github.com/chain4travel/camino-messenger-bot/internal/tracing"

	"buf.build/gen/go/chain4travel/camino-messenger-protocol/grpc/go/cmp/services/activity/v2/activityv2grpc"
	"buf.build/gen/go/chain4travel/camino-messenger-protocol/grpc/go/cmp/services/book/v2/bookv2grpc"
	"buf.build/gen/go/chain4travel/camino-messenger-protocol/grpc/go/cmp/services/info/v2/infov2grpc"
	"buf.build/gen/go/chain4travel/camino-messenger-protocol/grpc/go/cmp/services/insurance/v1/insurancev1grpc"
	"buf.build/gen/go/chain4travel/camino-messenger-protocol/grpc/go/cmp/services/seat_map/v2/seat_mapv2grpc"
	bookv2 "buf.build/gen/go/chain4travel/camino-messenger-protocol/protocolbuffers/go/cmp/services/book/v2"
	insurancev1 "buf.build/gen/go/chain4travel/camino-messenger-protocol/protocolbuffers/go/cmp/services/insurance/v1"
	seat_mapv2 "buf.build/gen/go/chain4travel/camino-messenger-protocol/protocolbuffers/go/cmp/services/seat_map/v2"

	"buf.build/gen/go/chain4travel/camino-messenger-protocol/grpc/go/cmp/services/accommodation/v2/accommodationv2grpc"
	"buf.build/gen/go/chain4travel/camino-messenger-protocol/grpc/go/cmp/services/network/v1/networkv1grpc"
	"buf.build/gen/go/chain4travel/camino-messenger-protocol/grpc/go/cmp/services/partner/v2/partnerv2grpc"
	"buf.build/gen/go/chain4travel/camino-messenger-protocol/grpc/go/cmp/services/ping/v1/pingv1grpc"
	"buf.build/gen/go/chain4travel/camino-messenger-protocol/grpc/go/cmp/services/transport/v2/transportv2grpc"
	accommodationv2 "buf.build/gen/go/chain4travel/camino-messenger-protocol/protocolbuffers/go/cmp/services/accommodation/v2"
	activityv2 "buf.build/gen/go/chain4travel/camino-messenger-protocol/protocolbuffers/go/cmp/services/activity/v2"
	infov2 "buf.build/gen/go/chain4travel/camino-messenger-protocol/protocolbuffers/go/cmp/services/info/v2"
	networkv1 "buf.build/gen/go/chain4travel/camino-messenger-protocol/protocolbuffers/go/cmp/services/network/v1"
	pingv1 "buf.build/gen/go/chain4travel/camino-messenger-protocol/protocolbuffers/go/cmp/services/ping/v1"
	transportv2 "buf.build/gen/go/chain4travel/camino-messenger-protocol/protocolbuffers/go/cmp/services/transport/v2"
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

	_ accommodationv2grpc.AccommodationProductInfoServiceServer = (*server)(nil)
	_ accommodationv2grpc.AccommodationProductListServiceServer = (*server)(nil)
	_ accommodationv2grpc.AccommodationSearchServiceServer      = (*server)(nil)
	_ activityv2grpc.ActivityProductListServiceServer           = (*server)(nil)
	_ activityv2grpc.ActivitySearchServiceServer                = (*server)(nil)
	_ networkv1grpc.GetNetworkFeeServiceServer                  = (*server)(nil)
	_ partnerv2grpc.GetPartnerConfigurationServiceServer        = (*server)(nil)
	_ bookv2grpc.MintServiceServer                              = (*server)(nil)
	_ bookv2grpc.ValidationServiceServer                        = (*server)(nil)
	_ pingv1grpc.PingServiceServer                              = (*server)(nil)
	_ transportv2grpc.TransportSearchServiceServer              = (*server)(nil)
	_ seat_mapv2grpc.SeatMapServiceServer                       = (*server)(nil)
	_ seat_mapv2grpc.SeatMapAvailabilityServiceServer           = (*server)(nil)
	_ infov2grpc.CountryEntryRequirementsServiceServer          = (*server)(nil)
	_ activityv2grpc.ActivityProductInfoServiceServer           = (*server)(nil)
	_ insurancev1grpc.InsuranceProductInfoServiceServer         = (*server)(nil)
	_ insurancev1grpc.InsuranceProductListServiceServer         = (*server)(nil)
	_ insurancev1grpc.InsuranceSearchServiceServer              = (*server)(nil)
)

type externalRequestProcessor interface {
	processExternalRequest(ctx context.Context, requestType messaging.MessageType, request *messaging.RequestContent) (*messaging.ResponseContent, error)
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
	activityv2grpc.RegisterActivityProductListServiceServer(grpcServer, server)
	activityv2grpc.RegisterActivitySearchServiceServer(grpcServer, server)
	accommodationv2grpc.RegisterAccommodationProductInfoServiceServer(grpcServer, server)
	accommodationv2grpc.RegisterAccommodationProductListServiceServer(grpcServer, server)
	accommodationv2grpc.RegisterAccommodationSearchServiceServer(grpcServer, server)
	networkv1grpc.RegisterGetNetworkFeeServiceServer(grpcServer, server)
	// partnerv2grpc.RegisterGetPartnerConfigurationServiceServer(grpcServer, server)
	bookv2grpc.RegisterMintServiceServer(grpcServer, server)
	bookv2grpc.RegisterValidationServiceServer(grpcServer, server)
	pingv1grpc.RegisterPingServiceServer(grpcServer, server)
	transportv2grpc.RegisterTransportSearchServiceServer(grpcServer, server)
	seat_mapv2grpc.RegisterSeatMapServiceServer(grpcServer, server)
	seat_mapv2grpc.RegisterSeatMapAvailabilityServiceServer(grpcServer, server)
	infov2grpc.RegisterCountryEntryRequirementsServiceServer(grpcServer, server)
	activityv2grpc.RegisterActivityProductInfoServiceServer(grpcServer, server)
	insurancev1grpc.RegisterInsuranceProductInfoServiceServer(grpcServer, server)
	insurancev1grpc.RegisterInsuranceProductListServiceServer(grpcServer, server)
	insurancev1grpc.RegisterInsuranceSearchServiceServer(grpcServer, server)
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

func (s *server) AccommodationProductInfo(ctx context.Context, request *accommodationv2.AccommodationProductInfoRequest) (*accommodationv2.AccommodationProductInfoResponse, error) {
	response, err := s.processExternalRequest(ctx, messaging.AccommodationProductInfoRequest, &messaging.RequestContent{AccommodationProductInfoRequest: request})
	return response.AccommodationProductInfoResponse, err
}

func (s *server) AccommodationProductList(ctx context.Context, request *accommodationv2.AccommodationProductListRequest) (*accommodationv2.AccommodationProductListResponse, error) {
	response, err := s.processExternalRequest(ctx, messaging.AccommodationProductListRequest, &messaging.RequestContent{AccommodationProductListRequest: request})
	return response.AccommodationProductListResponse, err
}

func (s *server) AccommodationSearch(ctx context.Context, request *accommodationv2.AccommodationSearchRequest) (*accommodationv2.AccommodationSearchResponse, error) {
	response, err := s.processExternalRequest(ctx, messaging.AccommodationSearchRequest, &messaging.RequestContent{AccommodationSearchRequest: request})
	return response.AccommodationSearchResponse, err // TODO set specific errors according to https://grpc.github.io/grpc/core/md_doc_statuscodes.html ?
}

func (s *server) Ping(ctx context.Context, request *pingv1.PingRequest) (*pingv1.PingResponse, error) {
	response, err := s.processExternalRequest(ctx, messaging.PingRequest, &messaging.RequestContent{PingRequest: request})
	return response.PingResponse, err
}

func (s *server) GetNetworkFee(ctx context.Context, request *networkv1.GetNetworkFeeRequest) (*networkv1.GetNetworkFeeResponse, error) {
	response, err := s.processInternalRequest(ctx, messaging.GetNetworkFeeRequest, &messaging.RequestContent{GetNetworkFeeRequest: request})
	return response.GetNetworkFeeResponse, err
}

// func (s *server) GetPartnerConfiguration(ctx context.Context, request *partnerv2.GetPartnerConfigurationRequest) (*partnerv2.GetPartnerConfigurationResponse, error) {
// 	response, err := s.processInternalRequest(ctx, messaging.GetPartnerConfigurationRequest, &messaging.RequestContent{GetPartnerConfigurationRequest: request})
// 	return response.GetPartnerConfigurationResponse, err
// }

func (s *server) ActivityProductInfo(ctx context.Context, request *activityv2.ActivityProductInfoRequest) (*activityv2.ActivityProductInfoResponse, error) {
	response, err := s.processExternalRequest(ctx, messaging.ActivityProductInfoRequest, &messaging.RequestContent{ActivityProductInfoRequest: request})
	return response.ActivityProductInfoResponse, err
}

func (s *server) ActivityProductList(ctx context.Context, request *activityv2.ActivityProductListRequest) (*activityv2.ActivityProductListResponse, error) {
	response, err := s.processExternalRequest(ctx, messaging.ActivityProductListRequest, &messaging.RequestContent{ActivityProductListRequest: request})
	return response.ActivityProductListResponse, err
}

func (s *server) ActivitySearch(ctx context.Context, request *activityv2.ActivitySearchRequest) (*activityv2.ActivitySearchResponse, error) {
	response, err := s.processExternalRequest(ctx, messaging.ActivitySearchRequest, &messaging.RequestContent{ActivitySearchRequest: request})
	return response.ActivitySearchResponse, err
}

func (s *server) Mint(ctx context.Context, request *bookv2.MintRequest) (*bookv2.MintResponse, error) {
	response, err := s.processExternalRequest(ctx, messaging.MintRequest, &messaging.RequestContent{MintRequest: request})
	return response.MintResponse, err
}

func (s *server) Validation(ctx context.Context, request *bookv2.ValidationRequest) (*bookv2.ValidationResponse, error) {
	response, err := s.processExternalRequest(ctx, messaging.ValidationRequest, &messaging.RequestContent{ValidationRequest: request})
	return response.ValidationResponse, err
}

func (s *server) TransportSearch(ctx context.Context, request *transportv2.TransportSearchRequest) (*transportv2.TransportSearchResponse, error) {
	response, err := s.processExternalRequest(ctx, messaging.TransportSearchRequest, &messaging.RequestContent{TransportSearchRequest: request})
	return response.TransportSearchResponse, err
}

func (s *server) SeatMap(ctx context.Context, request *seat_mapv2.SeatMapRequest) (*seat_mapv2.SeatMapResponse, error) {
	response, err := s.processExternalRequest(ctx, messaging.SeatMapRequest, &messaging.RequestContent{SeatMapRequest: request})
	return response.SeatMapResponse, err
}

func (s *server) SeatMapAvailability(ctx context.Context, request *seat_mapv2.SeatMapAvailabilityRequest) (*seat_mapv2.SeatMapAvailabilityResponse, error) {
	response, err := s.processExternalRequest(ctx, messaging.SeatMapAvailabilityRequest, &messaging.RequestContent{SeatMapAvailabilityRequest: request})
	return response.SeatMapAvailabilityResponse, err
}

func (s *server) CountryEntryRequirements(ctx context.Context, request *infov2.CountryEntryRequirementsRequest) (*infov2.CountryEntryRequirementsResponse, error) {
	response, err := s.processExternalRequest(ctx, messaging.CountryEntryRequirementsRequest, &messaging.RequestContent{CountryEntryRequirementsRequest: request})
	return response.CountryEntryRequirementsResponse, err
}

func (s *server) InsuranceProductInfo(ctx context.Context, request *insurancev1.InsuranceProductInfoRequest) (*insurancev1.InsuranceProductInfoResponse, error) {
	response, err := s.processExternalRequest(ctx, messaging.InsuranceProductInfoRequest, &messaging.RequestContent{InsuranceProductInfoRequest: request})
	return response.InsuranceProductInfoResponse, err
}

func (s *server) InsuranceProductList(ctx context.Context, request *insurancev1.InsuranceProductListRequest) (*insurancev1.InsuranceProductListResponse, error) {
	response, err := s.processExternalRequest(ctx, messaging.InsuranceProductListRequest, &messaging.RequestContent{InsuranceProductListRequest: request})
	return response.InsuranceProductListResponse, err
}

func (s *server) InsuranceSearch(ctx context.Context, request *insurancev1.InsuranceSearchRequest) (*insurancev1.InsuranceSearchResponse, error) {
	response, err := s.processExternalRequest(ctx, messaging.InsuranceSearchRequest, &messaging.RequestContent{InsuranceSearchRequest: request})
	return response.InsuranceSearchResponse, err
}

func (s *server) processInternalRequest(ctx context.Context, requestType messaging.MessageType, request *messaging.RequestContent) (*messaging.ResponseContent, error) {
	ctx, span := s.tracer.Start(ctx, "server.processInternalRequest", trace.WithSpanKind(trace.SpanKindServer))
	defer span.End()
	service, registered := s.serviceRegistry.GetService(requestType)
	if !registered {
		return nil, fmt.Errorf("%w: %s", messaging.ErrUnsupportedService, requestType)
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
