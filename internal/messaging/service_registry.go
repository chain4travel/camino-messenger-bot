/*
 * Copyright (C) 2022-2023, Chain4Travel AG. All rights reserved.
 * See the file LICENSE for licensing terms.
 */

package messaging

import (
	"sync"

	"buf.build/gen/go/chain4travel/camino-messenger-protocol/grpc/go/cmp/services/accommodation/v1/accommodationv1grpc"
	"buf.build/gen/go/chain4travel/camino-messenger-protocol/grpc/go/cmp/services/activity/v1/activityv1grpc"
	"buf.build/gen/go/chain4travel/camino-messenger-protocol/grpc/go/cmp/services/book/v1/bookv1grpc"
	"buf.build/gen/go/chain4travel/camino-messenger-protocol/grpc/go/cmp/services/info/v1/infov1grpc"
	"buf.build/gen/go/chain4travel/camino-messenger-protocol/grpc/go/cmp/services/seat_map/v1/seat_mapv1grpc"
	"buf.build/gen/go/chain4travel/camino-messenger-protocol/grpc/go/cmp/services/transport/v1/transportv1grpc"
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
			c := activityv1grpc.NewActivityProductListServiceClient(rpcClient.ClientConn)
			service = activityProductListService{client: c}
		case ActivitySearchRequest:
			c := activityv1grpc.NewActivitySearchServiceClient(rpcClient.ClientConn)
			service = activityService{client: &c}
		case AccommodationProductInfoRequest:
			c := accommodationv1grpc.NewAccommodationProductInfoServiceClient(rpcClient.ClientConn)
			service = accommodationProductInfoService{client: &c}
		case AccommodationProductListRequest:
			c := accommodationv1grpc.NewAccommodationProductListServiceClient(rpcClient.ClientConn)
			service = accommodationProductListService{client: &c}
		case AccommodationSearchRequest:
			c := accommodationv1grpc.NewAccommodationSearchServiceClient(rpcClient.ClientConn)
			service = accommodationService{client: &c}
		case GetNetworkFeeRequest:
			service = networkService{} // this service does not talk to partner plugin
		case GetPartnerConfigurationRequest:
			service = partnerService{} // this service does not talk to partner plugin
		case MintRequest:
			c := bookv1grpc.NewMintServiceClient(rpcClient.ClientConn)
			service = mintService{client: &c}
		case ValidationRequest:
			c := bookv1grpc.NewValidationServiceClient(rpcClient.ClientConn)
			service = validationService{client: &c}
		case PingRequest:
			service = pingService{} // this service does not talk to partner plugin
		case TransportSearchRequest:
			c := transportv1grpc.NewTransportSearchServiceClient(rpcClient.ClientConn)
			service = transportService{client: &c}
		case SeatMapRequest:
			c := seat_mapv1grpc.NewSeatMapServiceClient(rpcClient.ClientConn)
			service = seatMapService{client: &c}
		case SeatMapAvailabilityRequest:
			c := seat_mapv1grpc.NewSeatMapAvailabilityServiceClient(rpcClient.ClientConn)
			service = seatMapAvailabilityService{client: &c}
		case CountryEntryRequirementsRequest:
			c := infov1grpc.NewCountryEntryRequirementsServiceClient(rpcClient.ClientConn)
			service = countryEntryRequirementsService{client: &c}
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
