package server

import (
	"context"
	"fmt"
	"net"

	"github.com/chain4travel/camino-messenger-bot/internal/tracing"

	"buf.build/gen/go/chain4travel/camino-messenger-protocol/grpc/go/cmp/services/activity/v1/activityv1grpc"
	"buf.build/gen/go/chain4travel/camino-messenger-protocol/grpc/go/cmp/services/book/v1/bookv1grpc"
	"buf.build/gen/go/chain4travel/camino-messenger-protocol/grpc/go/cmp/services/info/v1/infov1grpc"
	"buf.build/gen/go/chain4travel/camino-messenger-protocol/grpc/go/cmp/services/insurance/v1/insurancev1grpc"
	"buf.build/gen/go/chain4travel/camino-messenger-protocol/grpc/go/cmp/services/seat_map/v1/seat_mapv1grpc"
	bookv1 "buf.build/gen/go/chain4travel/camino-messenger-protocol/protocolbuffers/go/cmp/services/book/v1"
	insurancev1 "buf.build/gen/go/chain4travel/camino-messenger-protocol/protocolbuffers/go/cmp/services/insurance/v1"
	seat_mapv1 "buf.build/gen/go/chain4travel/camino-messenger-protocol/protocolbuffers/go/cmp/services/seat_map/v1"

	"buf.build/gen/go/chain4travel/camino-messenger-protocol/grpc/go/cmp/services/accommodation/v1/accommodationv1grpc"
	"buf.build/gen/go/chain4travel/camino-messenger-protocol/grpc/go/cmp/services/network/v1/networkv1grpc"
	"buf.build/gen/go/chain4travel/camino-messenger-protocol/grpc/go/cmp/services/partner/v1/partnerv1grpc"
	"buf.build/gen/go/chain4travel/camino-messenger-protocol/grpc/go/cmp/services/ping/v1/pingv1grpc"
	"buf.build/gen/go/chain4travel/camino-messenger-protocol/grpc/go/cmp/services/transport/v1/transportv1grpc"
	accommodationv1 "buf.build/gen/go/chain4travel/camino-messenger-protocol/protocolbuffers/go/cmp/services/accommodation/v1"
	activityv1 "buf.build/gen/go/chain4travel/camino-messenger-protocol/protocolbuffers/go/cmp/services/activity/v1"
	infov1 "buf.build/gen/go/chain4travel/camino-messenger-protocol/protocolbuffers/go/cmp/services/info/v1"
	networkv1 "buf.build/gen/go/chain4travel/camino-messenger-protocol/protocolbuffers/go/cmp/services/network/v1"
	partnerv1 "buf.build/gen/go/chain4travel/camino-messenger-protocol/protocolbuffers/go/cmp/services/partner/v1"
	pingv1 "buf.build/gen/go/chain4travel/camino-messenger-protocol/protocolbuffers/go/cmp/services/ping/v1"
	transportv1 "buf.build/gen/go/chain4travel/camino-messenger-protocol/protocolbuffers/go/cmp/services/transport/v1"
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

	_ accommodationv1grpc.AccommodationProductInfoServiceServer = (*server)(nil)
	_ accommodationv1grpc.AccommodationProductListServiceServer = (*server)(nil)
	_ accommodationv1grpc.AccommodationSearchServiceServer      = (*server)(nil)
	_ activityv1grpc.ActivityProductListServiceServer           = (*server)(nil)
	_ activityv1grpc.ActivitySearchServiceServer                = (*server)(nil)
	_ networkv1grpc.GetNetworkFeeServiceServer                  = (*server)(nil)
	_ partnerv1grpc.GetPartnerConfigurationServiceServer        = (*server)(nil)
	_ bookv1grpc.MintServiceServer                              = (*server)(nil)
	_ bookv1grpc.ValidationServiceServer                        = (*server)(nil)
	_ pingv1grpc.PingServiceServer                              = (*server)(nil)
	_ transportv1grpc.TransportSearchServiceServer              = (*server)(nil)
	_ seat_mapv1grpc.SeatMapServiceServer                       = (*server)(nil)
	_ seat_mapv1grpc.SeatMapAvailabilityServiceServer           = (*server)(nil)
	_ infov1grpc.CountryEntryRequirementsServiceServer          = (*server)(nil)
	_ activityv1grpc.ActivityProductInfoServiceServer           = (*server)(nil)
	_ insurancev1grpc.InsuranceProductInfoServiceServer         = (*server)(nil)
	_ insurancev1grpc.InsuranceProductListServiceServer         = (*server)(nil)
	_ insurancev1grpc.InsuranceSearchServiceServer              = (*server)(nil)
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
	activityv1grpc.RegisterActivityProductListServiceServer(grpcServer, server)
	activityv1grpc.RegisterActivitySearchServiceServer(grpcServer, server)
	accommodationv1grpc.RegisterAccommodationProductInfoServiceServer(grpcServer, server)
	accommodationv1grpc.RegisterAccommodationProductListServiceServer(grpcServer, server)
	accommodationv1grpc.RegisterAccommodationSearchServiceServer(grpcServer, server)
	networkv1grpc.RegisterGetNetworkFeeServiceServer(grpcServer, server)
	partnerv1grpc.RegisterGetPartnerConfigurationServiceServer(grpcServer, server)
	bookv1grpc.RegisterMintServiceServer(grpcServer, server)
	bookv1grpc.RegisterValidationServiceServer(grpcServer, server)
	pingv1grpc.RegisterPingServiceServer(grpcServer, server)
	transportv1grpc.RegisterTransportSearchServiceServer(grpcServer, server)
	seat_mapv1grpc.RegisterSeatMapServiceServer(grpcServer, server)
	seat_mapv1grpc.RegisterSeatMapAvailabilityServiceServer(grpcServer, server)
	infov1grpc.RegisterCountryEntryRequirementsServiceServer(grpcServer, server)
	activityv1grpc.RegisterActivityProductInfoServiceServer(grpcServer, server)
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

func (s *server) AccommodationProductInfo(ctx context.Context, request *accommodationv1.AccommodationProductInfoRequest) (*accommodationv1.AccommodationProductInfoResponse, error) {
	response, err := s.processExternalRequest(ctx, messaging.AccommodationProductInfoRequest, &messaging.RequestContent{AccommodationProductInfoRequest: request})
	return response.AccommodationProductInfoResponse, err
}

func (s *server) AccommodationProductList(ctx context.Context, request *accommodationv1.AccommodationProductListRequest) (*accommodationv1.AccommodationProductListResponse, error) {
	response, err := s.processExternalRequest(ctx, messaging.AccommodationProductListRequest, &messaging.RequestContent{AccommodationProductListRequest: request})
	return response.AccommodationProductListResponse, err
}

func (s *server) AccommodationSearch(ctx context.Context, request *accommodationv1.AccommodationSearchRequest) (*accommodationv1.AccommodationSearchResponse, error) {
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

func (s *server) GetPartnerConfiguration(ctx context.Context, request *partnerv1.GetPartnerConfigurationRequest) (*partnerv1.GetPartnerConfigurationResponse, error) {
	response, err := s.processInternalRequest(ctx, messaging.GetPartnerConfigurationRequest, &messaging.RequestContent{GetPartnerConfigurationRequest: request})
	return response.GetPartnerConfigurationResponse, err
}

func (s *server) ActivityProductInfo(ctx context.Context, request *activityv1.ActivityProductInfoRequest) (*activityv1.ActivityProductInfoResponse, error) {
	response, err := s.processExternalRequest(ctx, messaging.ActivityProductInfoRequest, &messaging.RequestContent{ActivityProductInfoRequest: request})
	return response.ActivityProductInfoResponse, err
}

func (s *server) ActivityProductList(ctx context.Context, request *activityv1.ActivityProductListRequest) (*activityv1.ActivityProductListResponse, error) {
	response, err := s.processExternalRequest(ctx, messaging.ActivityProductListRequest, &messaging.RequestContent{ActivityProductListRequest: request})
	return response.ActivityProductListResponse, err
}

func (s *server) ActivitySearch(ctx context.Context, request *activityv1.ActivitySearchRequest) (*activityv1.ActivitySearchResponse, error) {
	response, err := s.processExternalRequest(ctx, messaging.ActivitySearchRequest, &messaging.RequestContent{ActivitySearchRequest: request})
	return response.ActivitySearchResponse, err
}

func (s *server) Mint(ctx context.Context, request *bookv1.MintRequest) (*bookv1.MintResponse, error) {
	response, err := s.processExternalRequest(ctx, messaging.MintRequest, &messaging.RequestContent{MintRequest: request})
	return response.MintResponse, err
}

func (s *server) Validation(ctx context.Context, request *bookv1.ValidationRequest) (*bookv1.ValidationResponse, error) {
	response, err := s.processExternalRequest(ctx, messaging.ValidationRequest, &messaging.RequestContent{ValidationRequest: request})
	return response.ValidationResponse, err
}

func (s *server) TransportSearch(ctx context.Context, request *transportv1.TransportSearchRequest) (*transportv1.TransportSearchResponse, error) {
	response, err := s.processExternalRequest(ctx, messaging.TransportSearchRequest, &messaging.RequestContent{TransportSearchRequest: request})
	return response.TransportSearchResponse, err
}

func (s *server) SeatMap(ctx context.Context, request *seat_mapv1.SeatMapRequest) (*seat_mapv1.SeatMapResponse, error) {
	response, err := s.processExternalRequest(ctx, messaging.SeatMapRequest, &messaging.RequestContent{SeatMapRequest: request})
	return response.SeatMapResponse, err
}

func (s *server) SeatMapAvailability(ctx context.Context, request *seat_mapv1.SeatMapAvailabilityRequest) (*seat_mapv1.SeatMapAvailabilityResponse, error) {
	response, err := s.processExternalRequest(ctx, messaging.SeatMapAvailabilityRequest, &messaging.RequestContent{SeatMapAvailabilityRequest: request})
	return response.SeatMapAvailabilityResponse, err
}

func (s *server) CountryEntryRequirements(ctx context.Context, request *infov1.CountryEntryRequirementsRequest) (*infov1.CountryEntryRequirementsResponse, error) {
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
