/*
 * Copyright (C) 2022-2023, Chain4Travel AG. All rights reserved.
 * See the file LICENSE for licensing terms.
 */

package messaging

import (
	"sync"

	"buf.build/gen/go/chain4travel/camino-messenger-protocol/grpc/go/cmp/services/accommodation/v1alpha/accommodationv1alphagrpc"
	"buf.build/gen/go/chain4travel/camino-messenger-protocol/grpc/go/cmp/services/activity/v1alpha/activityv1alphagrpc"
	"buf.build/gen/go/chain4travel/camino-messenger-protocol/grpc/go/cmp/services/book/v1alpha/bookv1alphagrpc"
	"buf.build/gen/go/chain4travel/camino-messenger-protocol/grpc/go/cmp/services/seat_map/v1alpha/seat_mapv1alphagrpc"
	"buf.build/gen/go/chain4travel/camino-messenger-protocol/grpc/go/cmp/services/transport/v1alpha/transportv1alphagrpc"
	"github.com/chain4travel/camino-messenger-bot/config"
	"github.com/chain4travel/camino-messenger-bot/internal/rpc/client"
	"go.uber.org/zap"
)

type ServiceRegistry interface {
	RegisterServices(requestTypes config.SupportedRequestTypesFlag, rpcClient *client.RPCClient)
	GetService(messageType MessageType) (Service, bool)
}
type serviceRegistry struct {
	logger   *zap.SugaredLogger
	services map[MessageType]Service
	lock     *sync.RWMutex
}

func NewServiceRegistry(logger *zap.SugaredLogger) ServiceRegistry {
	return &serviceRegistry{
		logger:   logger,
		services: make(map[MessageType]Service),
		lock:     &sync.RWMutex{},
	}
}

func (s *serviceRegistry) RegisterServices(requestTypes config.SupportedRequestTypesFlag, rpcClient *client.RPCClient) {
	s.lock.Lock()
	defer s.lock.Unlock()

	for _, requestType := range requestTypes {
		var service Service
		switch MessageType(requestType) {
		case ActivityProductListRequest:
			c := activityv1alphagrpc.NewActivityProductListServiceClient(rpcClient.ClientConn)
			service = activityProductListService{client: c}
		case ActivitySearchRequest:
			c := activityv1alphagrpc.NewActivitySearchServiceClient(rpcClient.ClientConn)
			service = activityService{client: &c}
		case AccommodationProductInfoRequest:
			c := accommodationv1alphagrpc.NewAccommodationProductInfoServiceClient(rpcClient.ClientConn)
			service = accommodationProductInfoService{client: &c}
		case AccommodationProductListRequest:
			c := accommodationv1alphagrpc.NewAccommodationProductListServiceClient(rpcClient.ClientConn)
			service = accommodationProductListService{client: &c}
		case AccommodationSearchRequest:
			c := accommodationv1alphagrpc.NewAccommodationSearchServiceClient(rpcClient.ClientConn)
			service = accommodationService{client: &c}
		case GetNetworkFeeRequest:
			service = networkService{} // this service does not talk to partner plugin
		case GetPartnerConfigurationRequest:
			service = partnerService{} // this service does not talk to partner plugin
		case MintRequest:
			c := bookv1alphagrpc.NewMintServiceClient(rpcClient.ClientConn)
			service = mintService{client: &c}
		case ValidationRequest:
			c := bookv1alphagrpc.NewValidationServiceClient(rpcClient.ClientConn)
			service = validationService{client: &c}
		case PingRequest:
			service = pingService{} // this service does not talk to partner plugin
		case TransportSearchRequest:
			c := transportv1alphagrpc.NewTransportSearchServiceClient(rpcClient.ClientConn)
			service = transportService{client: &c}
		case SeatMapRequest:
			c := seat_mapv1alphagrpc.NewSeatMapServiceClient(rpcClient.ClientConn)
			service = seat_mapService{client: &c}
		default:
			s.logger.Infof("Skipping registration of unknown request type: %s", requestType)
			continue
		}
		s.services[MessageType(requestType)] = service
	}
}

func (s *serviceRegistry) GetService(messageType MessageType) (Service, bool) {
	service, ok := s.services[messageType]
	return service, ok
}
