package server

import (
	"context"

	"buf.build/gen/go/chain4travel/camino-messenger-protocol/grpc/go/cmp/services/partner/v1/partnerv1grpc"
	partnerv1 "buf.build/gen/go/chain4travel/camino-messenger-protocol/protocolbuffers/go/cmp/services/partner/v1"

	"github.com/chain4travel/camino-messenger-bot/internal/messaging"
	"google.golang.org/grpc"
)

var (
	_ partnerv1grpc.GetPartnerConfigurationServiceServer = (*partner_srv1)(nil)
)

type partner_srv1 struct {
	reqProcessor externalRequestProcessor
}

func NewPartnerSrv1(
	grpcServer *grpc.Server,
	reqProcess externalRequestProcessor,
) partnerv1grpc.GetPartnerConfigurationServiceServer {
	partner_srv := &partner_srv1{reqProcessor: reqProcess}
	partnerv1grpc.RegisterGetPartnerConfigurationServiceServer(grpcServer, partner_srv)
	return partner_srv
}

func (s *partner_srv1) GetPartnerConfiguration(ctx context.Context, request *partnerv1.GetPartnerConfigurationRequest) (*partnerv1.GetPartnerConfigurationResponse, error) {
	response, err := s.reqProcessor.processExternalRequest(
		ctx,
		messaging.PingRequest, request,
	)
	return response.(*partnerv1.GetPartnerConfigurationResponse), err
}
