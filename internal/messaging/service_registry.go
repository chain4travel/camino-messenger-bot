/*
 * Copyright (C) 2022-2023, Chain4Travel AG. All rights reserved.
 * See the file LICENSE for licensing terms.
 */

package messaging

import (
	"buf.build/gen/go/chain4travel/camino-messenger-protocol/grpc/go/cmp/services/accommodation/v1alpha/accommodationv1alphagrpc"
	"buf.build/gen/go/chain4travel/camino-messenger-protocol/grpc/go/cmp/services/activity/v1alpha/activityv1alphagrpc"
	"buf.build/gen/go/chain4travel/camino-messenger-protocol/grpc/go/cmp/services/ping/v1alpha/pingv1alphagrpc"
	"buf.build/gen/go/chain4travel/camino-messenger-protocol/grpc/go/cmp/services/transport/v1alpha/transportv1alphagrpc"
	"github.com/chain4travel/camino-messenger-bot/config"
	"github.com/chain4travel/camino-messenger-bot/internal/rpc/client"
	"go.uber.org/zap"
)

type ServiceRegistry struct {
	logger    *zap.SugaredLogger
	rpcClient *client.RPCClient
	services  map[MessageType]Service
}

func NewServiceRegistry(logger *zap.SugaredLogger, rpcClient *client.RPCClient) *ServiceRegistry {
	return &ServiceRegistry{
		logger:    logger,
		rpcClient: rpcClient,
		services:  make(map[MessageType]Service),
	}
}

func (s *ServiceRegistry) RegisterServices(requestTypes config.SupportedRequestTypesFlag) {
	for _, requestType := range requestTypes {
		var service Service
		switch MessageType(requestType) {
		case ActivitySearchRequest:
			c := activityv1alphagrpc.NewActivitySearchServiceClient(s.rpcClient.ClientConn)
			service = activityService{client: &c}
		case AccommodationProductInfoRequest:
			c := accommodationv1alphagrpc.NewAccommodationProductInfoServiceClient(s.rpcClient.ClientConn)
			service = accommodationProductInfoService{client: &c}
		case AccommodationProductListRequest:
			c := accommodationv1alphagrpc.NewAccommodationProductListServiceClient(s.rpcClient.ClientConn)
			service = accommodationProductListService{client: &c}
		case AccommodationSearchRequest:
			c := accommodationv1alphagrpc.NewAccommodationSearchServiceClient(s.rpcClient.ClientConn)
			service = accommodationService{client: &c}
		case GetNetworkFeeRequest:
			service = networkService{} // this service does not talk to partner plugin
		case GetPartnerConfigurationRequest:
			service = partnerService{} // this service does not talk to partner plugin
		case PingRequest:
			c := pingv1alphagrpc.NewPingServiceClient(s.rpcClient.ClientConn)
			service = pingService{client: &c}
		case TransportSearchRequest:
			c := transportv1alphagrpc.NewTransportSearchServiceClient(s.rpcClient.ClientConn)
			service = transportService{client: &c}
		default:
			s.logger.Infof("Skipping registration of unknown request type: %s", requestType)
			continue
		}
		s.services[MessageType(requestType)] = service
	}
}

func (s *ServiceRegistry) GetService(messageType MessageType) (Service, bool) {
	service, ok := s.services[messageType]
	return service, ok
}
