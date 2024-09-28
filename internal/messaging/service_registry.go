/*
 * Copyright (C) 2022-2023, Chain4Travel AG. All rights reserved.
 * See the file LICENSE for licensing terms.
 */
package messaging

import (
	"strconv"
	"strings"
	"sync"

	"buf.build/gen/go/chain4travel/camino-messenger-protocol/grpc/go/cmp/services/accommodation/v2/accommodationv2grpc"
	"buf.build/gen/go/chain4travel/camino-messenger-protocol/grpc/go/cmp/services/activity/v2/activityv2grpc"
	"buf.build/gen/go/chain4travel/camino-messenger-protocol/grpc/go/cmp/services/book/v2/bookv2grpc"
	"buf.build/gen/go/chain4travel/camino-messenger-protocol/grpc/go/cmp/services/info/v2/infov2grpc"
	"buf.build/gen/go/chain4travel/camino-messenger-protocol/grpc/go/cmp/services/insurance/v1/insurancev1grpc"
	"buf.build/gen/go/chain4travel/camino-messenger-protocol/grpc/go/cmp/services/seat_map/v2/seat_mapv2grpc"
	"buf.build/gen/go/chain4travel/camino-messenger-protocol/grpc/go/cmp/services/transport/v2/transportv2grpc"
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
	logger    *zap.SugaredLogger
	services  map[MessageType]Service
	lock      *sync.RWMutex
	supported map[ServiceIdentifier]cmaccount.PartnerConfigurationService
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
		serviceVersion, err := strconv.ParseUint(servicePath[3][1:], 10, 64)
		if err != nil {
			logger.Errorf("Error:", err)
		}
		logger.Info(servicePath[4], " registered version:", serviceVersion)
		supported[ServiceIdentifier{serviceName: servicePath[4], serviceVersion: serviceVersion, servicePath: serviceFullName}] = supportedServices.Services[i]
	}

	return &serviceRegistry{
		logger:    logger,
		services:  make(map[MessageType]Service),
		lock:      &sync.RWMutex{},
		supported: supported,
	}
}

func (s *serviceRegistry) RegisterServices(rpcClient *client.RPCClient) {
	s.lock.Lock()
	defer s.lock.Unlock()

	if s.isServiceVersionSupported("ActivityProductInfoService", uint64(2), "cmp.services.activity.v2.ActivityProductInfoService") {
		c := activityv2grpc.NewActivityProductInfoServiceClient(rpcClient.ClientConn)
		s.services[MessageType(s.getRequestTypeNameFromServiceName("ActivityProductInfoService"))] = activityProductInfoService{client: &c}
	}
	if s.isServiceVersionSupported("ActivityProductListService", uint64(2), "cmp.services.activity.v2.ActivityProductListService") {
		c := activityv2grpc.NewActivityProductListServiceClient(rpcClient.ClientConn)
		s.services[MessageType(s.getRequestTypeNameFromServiceName("ActivityProductListService"))] = activityProductListService{client: c}
	}
	if s.isServiceVersionSupported("ActivitySearchService", uint64(2), "cmp.services.activity.v2.ActivitySearchService'") {
		c := activityv2grpc.NewActivitySearchServiceClient(rpcClient.ClientConn)
		s.services[MessageType(s.getRequestTypeNameFromServiceName("ActivitySearchService"))] = activityService{client: &c}
	}
	if s.isServiceVersionSupported("AccommodationProductInfoService", uint64(2), "cmp.services.accommodation.v2.AccommodationProductInfoService") {
		c := accommodationv2grpc.NewAccommodationProductInfoServiceClient(rpcClient.ClientConn)
		s.services[MessageType(s.getRequestTypeNameFromServiceName("AccommodationProductInfoService"))] = accommodationProductInfoService{client: &c}
	}
	if s.isServiceVersionSupported("AccommodationProductListService", uint64(2), "cmp.services.accommodation.v2.AccommodationProductListService") {
		c := accommodationv2grpc.NewAccommodationProductListServiceClient(rpcClient.ClientConn)
		s.services[MessageType(s.getRequestTypeNameFromServiceName("AccommodationProductListService"))] = accommodationProductListService{client: &c}
	}
	if s.isServiceVersionSupported("AccommodationSearchService", uint64(2), "cmp.services.accommodation.v2.AccommodationSearchService") {
		c := accommodationv2grpc.NewAccommodationSearchServiceClient(rpcClient.ClientConn)
		s.services[MessageType(s.getRequestTypeNameFromServiceName("AccommodationSearchService"))] = accommodationSearchService{client: &c}
	}
	if s.isServiceVersionSupported("MintService", uint64(2), "cmp.services.book.v2.MintService") {
		c := bookv2grpc.NewMintServiceClient(rpcClient.ClientConn)
		s.services[MessageType(s.getRequestTypeNameFromServiceName("MintService"))] = mintService{client: &c}
	}
	if s.isServiceVersionSupported("ValidationService", uint64(2), "cmp.services.book.v2.ValidationService") {
		c := bookv2grpc.NewValidationServiceClient(rpcClient.ClientConn)
		s.services[MessageType(s.getRequestTypeNameFromServiceName("ValidationService"))] = validationService{client: &c}
	}
	if s.isServiceVersionSupported("TransportSearchService", uint64(2), "cmp.services.transport.v2.TransportSearchService") {
		c := transportv2grpc.NewTransportSearchServiceClient(rpcClient.ClientConn)
		s.services[MessageType(s.getRequestTypeNameFromServiceName("TransportSearchService"))] = transportService{client: &c}
	}
	if s.isServiceVersionSupported("SeatMapService", uint64(2), "cmp.services.seat_map.v2.SeatMapService") {
		c := seat_mapv2grpc.NewSeatMapServiceClient(rpcClient.ClientConn)
		s.services[MessageType(s.getRequestTypeNameFromServiceName("SeatMapService"))] = seatMapService{client: &c}
	}
	if s.isServiceVersionSupported("SeatMapAvailabilityService", uint64(2), "cmp.services.seat_map.v2.SeatMapAvailabilityService") {
		c := seat_mapv2grpc.NewSeatMapAvailabilityServiceClient(rpcClient.ClientConn)
		s.services[MessageType(s.getRequestTypeNameFromServiceName("SeatMapAvailabilityService"))] = seatMapAvailabilityService{client: &c}
	}
	if s.isServiceVersionSupported("CountryEntryRequirementsService", uint64(2), "cmp.services.info.v2.CountryEntryRequirementsService") {
		c := infov2grpc.NewCountryEntryRequirementsServiceClient(rpcClient.ClientConn)
		s.services[MessageType(s.getRequestTypeNameFromServiceName("CountryEntryRequirementsService"))] = countryEntryRequirementsService{client: &c}
	}
	if s.isServiceVersionSupported("InsuranceSearchService", uint64(1), "cmp.services.insurance.v1.InsuranceSearchService") {
		c := insurancev1grpc.NewInsuranceSearchServiceClient(rpcClient.ClientConn)
		s.services[MessageType(s.getRequestTypeNameFromServiceName("InsuranceSearchService"))] = insuranceSearchService{client: &c}
	}
	if s.isServiceVersionSupported("InsuranceProductListService", uint64(1), "cmp.services.insurance.v1.InsuranceProductListService") {
		c := insurancev1grpc.NewInsuranceProductListServiceClient(rpcClient.ClientConn)
		s.services[MessageType(s.getRequestTypeNameFromServiceName("InsuranceProductListService"))] = insuranceProductListService{client: &c}
	}
	if s.isServiceVersionSupported("GetNetworkFeeService", uint64(1), "cmp.services.network.1.GetNetworkFeeService") {
		s.services[MessageType(s.getRequestTypeNameFromServiceName("GetNetworkFeeService"))] = networkService{}
	}
	if s.isServiceVersionSupported("PingService", uint64(1), "cmp.services.ping.v1.PingService") {
		s.services[MessageType(s.getRequestTypeNameFromServiceName("PingService"))] = pingService{}
	}
	if s.isServiceVersionSupported("GetPartnerConfigurationService", uint64(2), "cmp.services.partner.v2.GetPartnerConfigurationService") {
		s.services[MessageType(s.getRequestTypeNameFromServiceName("GetPartnerConfigurationService"))] = partnerService{}
	}
}

func (s *serviceRegistry) getRequestTypeNameFromServiceName(name string) string {
	name = strings.TrimSuffix(name, "Service")
	return name + "Request"
}

func (s *serviceRegistry) GetService(messageType MessageType) (Service, bool) {
	s.lock.RLock()
	defer s.lock.RUnlock()
	service, ok := s.services[messageType]
	return service, ok
}

func (s *serviceRegistry) isServiceVersionSupported(name string, version uint64, path string) bool {
	_, ok := s.supported[ServiceIdentifier{serviceName: name, serviceVersion: version, servicePath: path}]
	return ok
}
