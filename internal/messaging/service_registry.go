/*
 * Copyright (C) 2022-2023, Chain4Travel AG. All rights reserved.
 * See the file LICENSE for licensing terms.
 */

package messaging

import (
	"log"
	"strconv"
	"strings"
	"sync"

	"buf.build/gen/go/chain4travel/camino-messenger-protocol/grpc/go/cmp/services/accommodation/v1/accommodationv1grpc"
	"buf.build/gen/go/chain4travel/camino-messenger-protocol/grpc/go/cmp/services/activity/v1/activityv1grpc"
	"buf.build/gen/go/chain4travel/camino-messenger-protocol/grpc/go/cmp/services/book/v1/bookv1grpc"
	"buf.build/gen/go/chain4travel/camino-messenger-protocol/grpc/go/cmp/services/info/v1/infov1grpc"
	"buf.build/gen/go/chain4travel/camino-messenger-protocol/grpc/go/cmp/services/seat_map/v1/seat_mapv1grpc"
	"buf.build/gen/go/chain4travel/camino-messenger-protocol/grpc/go/cmp/services/transport/v1/transportv1grpc"
	"github.com/chain4travel/camino-messenger-bot/internal/rpc/client"
	"github.com/chain4travel/camino-messenger-contracts/go/contracts/cmaccount"
	"go.uber.org/zap"
)

type ServiceRegistry interface {
	RegisterServices(rpcClient *client.RPCClient)
	GetService(messageType MessageType) (Service, bool)
}

type supportedServices struct {
	ServiceNames []string
	Services     []cmaccount.PartnerConfigurationService
}

type serviceRegistry struct {
	logger            *zap.SugaredLogger
	services          map[MessageType]Service
	supportedServices supportedServices
	lock              *sync.RWMutex
	supported         map[ServiceIdentifier]cmaccount.PartnerConfigurationService
}
type ServiceIdentifier struct {
	serviceName    string
	serviceVersion uint64
	servicePath    string
}

func NewServiceRegistry(supportedServices supportedServices, logger *zap.SugaredLogger) ServiceRegistry {
	supported := make(map[ServiceIdentifier]cmaccount.PartnerConfigurationService)
	// TODO: @VjeraTurk support multiple versions
	for i, serviceFullName := range supportedServices.ServiceNames {
		// Split each service name by "."
		servicePath := strings.Split(serviceFullName, ".")

		if len(servicePath) < 4 {
			logger.Errorf("Unidentified service: %s ", serviceFullName)
		}
		serviceVersion := servicePath[3]
		serviceVersion = serviceVersion[1:]
		num, err := strconv.ParseUint(serviceVersion, 10, 64)
		if err != nil {
			logger.Errorf("Error:", err)
		}
		logger.Info(servicePath[4], " registered version:", num)
		supported[ServiceIdentifier{serviceName: servicePath[4], serviceVersion: num, servicePath: serviceFullName}] = supportedServices.Services[i]
	}

	return &serviceRegistry{
		logger:            logger,
		services:          make(map[MessageType]Service),
		supportedServices: supportedServices,
		lock:              &sync.RWMutex{},
		supported:         supported,
	}
}

func (s *serviceRegistry) RegisterServices(rpcClient *client.RPCClient) {
	s.lock.Lock()
	defer s.lock.Unlock()

	requestTypes := []string{
		"GetNetworkFeeRequest",
		"GetPartnerConfigurationRequest",
		"PingRequest",
		"AccommodationProductListRequest",
		"AccommodationProductInfoRequest",
		"AccommodationSearchRequest",
		"ActivityProductListRequest",
		"ActivityProductInfoRequest",
		"ActivitySearchRequest",
		"TransportSearchRequest",
		"MintRequest",
		"ValidationRequest",
		"SeatMapRequest",
		"SeatMapAvailabilityRequest",
		"CountryEntryRequirementsRequest",
	}
	for _, requestType := range requestTypes {
		var service Service
		switch MessageType(requestType) {
		case ActivityProductInfoRequest:
			if s.isServiceVersionSupported("ActivityProductInfoService", uint64(1), "cmp.services.activity.v1.ActivityProductInfoService") {
				c := activityv1grpc.NewActivityProductInfoServiceClient(rpcClient.ClientConn)
				service = activityProductInfoService{client: &c}
			}
		case ActivityProductListRequest:
			if s.isServiceVersionSupported("ActivityProductListService", uint64(1), "cmp.services.activity.v1.ActivityProductInfoService") {
				c := activityv1grpc.NewActivityProductListServiceClient(rpcClient.ClientConn)
				service = activityProductListService{client: c}
			}
		case ActivitySearchRequest:
			if s.isServiceVersionSupported("ActivitySearchService", uint64(1), "cmp.services.activity.v1.ActivitySearchService'") {
				c := activityv1grpc.NewActivitySearchServiceClient(rpcClient.ClientConn)
				service = activityService{client: &c}
			}
		case AccommodationProductInfoRequest:
			if s.isServiceVersionSupported("AccommodationProductInfoService", uint64(1), "cmp.services.accommodation.v1.AccommodationProductInfoService") {
				c := accommodationv1grpc.NewAccommodationProductInfoServiceClient(rpcClient.ClientConn)
				service = accommodationProductInfoService{client: &c}
			}
		case AccommodationProductListRequest:
			if s.isServiceVersionSupported("AccommodationProductListService", uint64(1), "cmp.services.accommodation.v1.AccommodationProductInfoService") {
				c := accommodationv1grpc.NewAccommodationProductListServiceClient(rpcClient.ClientConn)
				service = accommodationProductListService{client: &c}
			}
		case AccommodationSearchRequest:
			if s.isServiceVersionSupported("AccommodationSearchService", uint64(1), "cmp.services.accommodation.v1.AccommodationSearchService") {
				c := accommodationv1grpc.NewAccommodationSearchServiceClient(rpcClient.ClientConn)
				service = accommodationSearchService{client: &c}
			}
		case MintRequest:
			if s.isServiceVersionSupported("MintService", uint64(1), "cmp.services.book.v1.MintService") {
				c := bookv1grpc.NewMintServiceClient(rpcClient.ClientConn)
				service = mintService{client: &c}
			}
			// WIP - support multiple version - otherwise linter complains
			if s.isServiceVersionSupported("MintService", uint64(2), "cmp.services.book.v2.MintService") {
				// c := bookv2grpc.NewMintServiceClient(rpcClient.ClientConn)
				// services[2] = mintService{client: &c}
				log.Print("supports MintService 2")
			}
		case ValidationRequest:
			if s.isServiceVersionSupported("ValidationService", uint64(1), "cmp.services.book.v1.ValidationService") {
				c := bookv1grpc.NewValidationServiceClient(rpcClient.ClientConn)
				service = validationService{client: &c}
			}
		case TransportSearchRequest:
			if s.isServiceVersionSupported("TransportSearchService", uint64(1), "cmp.services.transport.v1.TransportSearchService") {
				c := transportv1grpc.NewTransportSearchServiceClient(rpcClient.ClientConn)
				service = transportService{client: &c}
			}
		case SeatMapRequest:
			if s.isServiceVersionSupported("SeatMapService", uint64(1), "cmp.services.seat_map.v1.SeatMapService") {
				c := seat_mapv1grpc.NewSeatMapServiceClient(rpcClient.ClientConn)
				service = seatMapService{client: &c}
			}
		case SeatMapAvailabilityRequest:
			if s.isServiceVersionSupported("SeatMapAvailabilityService", uint64(1), "cmp.services.seat_map.v1.SeatMapAvailabilityService") {
				c := seat_mapv1grpc.NewSeatMapAvailabilityServiceClient(rpcClient.ClientConn)
				service = seatMapAvailabilityService{client: &c}
			}
		case CountryEntryRequirementsRequest:
			if s.isServiceVersionSupported("CountryEntryRequirementsService", uint64(1), "cmp.services.info.v1.CountryEntryRequirementsService") {
				c := infov1grpc.NewCountryEntryRequirementsServiceClient(rpcClient.ClientConn)
				service = countryEntryRequirementsService{client: &c}
			}
		case GetNetworkFeeRequest:
			service = networkService{} // this service does not talk to partner plugin
		case GetPartnerConfigurationRequest:
			service = partnerService{} // this service does not talk to partner plugin
		case PingRequest:
			service = pingService{} // this service does not talk to partner plugin
		default:
			s.logger.Infof("Skipping registration of unknown request type: %s", requestType)
			continue
		}
		if service != nil {
			s.services[MessageType(requestType)] = service
		}
	}
}

func (s *serviceRegistry) GetService(messageType MessageType) (Service, bool) {
	service, ok := s.services[messageType]
	return service, ok
}

func (s *serviceRegistry) isServiceVersionSupported(name string, version uint64, path string) bool {
	_, ok := s.supported[ServiceIdentifier{serviceName: name, serviceVersion: version, servicePath: path}]
	return ok
}

func (s *serviceRegistry) GetServiceByName(serviceName string) (cmaccount.PartnerConfigurationService, bool) {
	s.lock.RLock()
	defer s.lock.RUnlock()

	for identifier, service := range s.supported {
		if identifier.serviceName == serviceName {
			return service, true
		}
	}
	return cmaccount.PartnerConfigurationService{}, false
}
