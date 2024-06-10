package server

import (
	"context"

	"buf.build/gen/go/chain4travel/camino-messenger-protocol/grpc/go/cmp/services/partner/v2/partnerv2grpc"
	partnerv2 "buf.build/gen/go/chain4travel/camino-messenger-protocol/protocolbuffers/go/cmp/services/partner/v2"

	"github.com/chain4travel/camino-messenger-bot/internal/messaging"
	"google.golang.org/grpc"
)

var (
	_ partnerv2grpc.GetPartnerConfigurationServiceServer = (*partner_srv2)(nil)
)

type partner_srv2 struct {
	reqProcessor externalRequestProcessor
}

func NewPartnerSrv2(
	grpcServer *grpc.Server,
	reqProcess externalRequestProcessor,
) partnerv2grpc.GetPartnerConfigurationServiceServer {
	partner_srv := &partner_srv2{reqProcessor: reqProcess}
	partnerv2grpc.RegisterGetPartnerConfigurationServiceServer(grpcServer, partner_srv)
	return partner_srv
}

func (s *partner_srv2) GetPartnerConfiguration(ctx context.Context, request *partnerv2.GetPartnerConfigurationRequest) (*partnerv2.GetPartnerConfigurationResponse, error) {
	response, err := s.reqProcessor.processExternalRequest(
		ctx,
		messaging.PingRequest, &messaging.RequestContent{
			GetPartnerConfigurationRequestV2: request,
		},
	)
	return response.GetPartnerConfigurationResponseV2, err
}
