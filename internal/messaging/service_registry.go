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
	"buf.build/gen/go/chain4travel/camino-messenger-protocol/grpc/go/cmp/services/notification/v1/notificationv1grpc"
	"buf.build/gen/go/chain4travel/camino-messenger-protocol/grpc/go/cmp/services/seat_map/v2/seat_mapv2grpc"
	"buf.build/gen/go/chain4travel/camino-messenger-protocol/grpc/go/cmp/services/transport/v2/transportv2grpc"
	"github.com/chain4travel/camino-messenger-bot/internal/rpc/client"
	"github.com/chain4travel/camino-messenger-contracts/go/contracts/cmaccount"
	"go.uber.org/zap"
)

type ServiceRegistry interface {
	RegisterServices(rpcClient *client.RPCClient)
	GetService(messageType MessageType) (Service, bool)

	// should only be called for supplier bot with rpc client
	NotificationClient() notificationv1grpc.NotificationServiceClient
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
	rpcClient *client.RPCClient
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

	hasUnsupported := false
	// Bot does a simple check - if there is a service in the CM-Account which is not supported by the Bot it does not start up.
	if s.isServiceVersionSupported("ActivityProductInfoService", uint64(1), "cmp.services.activity.v1.ActivityProductInfoService") {
		s.logger.Infof("Service version is not supported: cmp.services.activity.v1.ActivityProductInfoService")
		hasUnsupported = true
	}
	if s.isServiceVersionSupported("ActivityProductListService", uint64(1), "cmp.services.activity.v1.ActivityProductListService") {
		s.logger.Infof("Service version is not supported: cmp.services.activity.v1.ActivityProductListService")
		hasUnsupported = true
	}
	if s.isServiceVersionSupported("ActivitySearchService", uint64(1), "cmp.services.activity.v1.ActivitySearchService") {
		s.logger.Infof("Service version is not supported: cmp.services.activity.v1.ActivitySearchService")
		hasUnsupported = true
	}
	if s.isServiceVersionSupported("AccommodationProductInfoService", uint64(1), "cmp.services.accommodation.v1.AccommodationProductInfoService") {
		s.logger.Infof("Service version is not supported: cmp.services.accommodation.v1.AccommodationProductInfoService")
		hasUnsupported = true
	}
	if s.isServiceVersionSupported("AccommodationProductListService", uint64(1), "cmp.services.accommodation.v1.AccommodationProductListService") {
		s.logger.Infof("Service version is not supported: cmp.services.accommodation.v1.AccommodationProductListService")
		hasUnsupported = true
	}
	if s.isServiceVersionSupported("AccommodationSearchService", uint64(1), "cmp.services.accommodation.v1.AccommodationSearchService") {
		s.logger.Infof("Service version is not supported: cmp.services.accommodation.v1.AccommodationSearchService")
		hasUnsupported = true
	}
	if s.isServiceVersionSupported("MintService", uint64(1), "cmp.services.book.v1.MintService") {
		s.logger.Infof("Service version is not supported: cmp.services.book.v1.MintService")
		hasUnsupported = true
	}
	if s.isServiceVersionSupported("ValidationService", uint64(1), "cmp.services.book.v1.ValidationService") {
		s.logger.Infof("Service version is not supported: cmp.services.book.v1.ValidationService")
		hasUnsupported = true
	}
	if s.isServiceVersionSupported("TransportSearchService", uint64(1), "cmp.services.transport.v1.TransportSearchService") {
		s.logger.Infof("Service version is not supported: cmp.services.transport.v1.TransportSearchService")
		hasUnsupported = true
	}
	if s.isServiceVersionSupported("SeatMapService", uint64(1), "cmp.services.seat_map.v1.SeatMapService") {
		s.logger.Infof("Service version is not supported: cmp.services.seat_map.v1.SeatMapService")
		hasUnsupported = true
	}
	if s.isServiceVersionSupported("SeatMapAvailabilityService", uint64(1), "cmp.services.seat_map.v1.SeatMapAvailabilityService") {
		s.logger.Infof("Service version is not supported: cmp.services.seat_map.v1.SeatMapAvailabilityService")
		hasUnsupported = true
	}
	if s.isServiceVersionSupported("CountryEntryRequirementsService", uint64(1), "cmp.services.info.v1.CountryEntryRequirementsService") {
		s.logger.Infof("Service version is not supported: cmp.services.info.v1.CountryEntryRequirementsService")
		hasUnsupported = true
	}

	if hasUnsupported {
		s.logger.Fatalf("Unsupported services detected. Please upgrade the service version.")
	}

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
	if s.isServiceVersionSupported("InsuranceProductInfoService", uint64(1), "cmp.services.insurance.v1.InsuranceProductInfoService") {
		c := insurancev1grpc.NewInsuranceProductInfoServiceClient(rpcClient.ClientConn)
		s.services[MessageType(s.getRequestTypeNameFromServiceName("InsuranceProductInfoService"))] = insuranceProductInfoService{client: &c}
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

	if rpcClient != nil {
		s.rpcClient = rpcClient
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

func (s *serviceRegistry) NotificationClient() notificationv1grpc.NotificationServiceClient {
	return notificationv1grpc.NewNotificationServiceClient(s.rpcClient.ClientConn)
}

var servicesMapping = map[MessageType]ServiceIdentifier{
	ActivityProductInfoRequest: {
		serviceName:    "ActivityProductInfoService",
		servicePath:    "cmp.services.activity.v2.ActivityProductInfoService",
		serviceVersion: 2,
	},
	ActivityProductListRequest: {
		serviceName:    "ActivityProductListService",
		servicePath:    "cmp.services.activity.v2.ActivityProductListService",
		serviceVersion: 2,
	},
	ActivitySearchRequest: {
		serviceName:    "ActivitySearchService",
		servicePath:    "cmp.services.activity.v2.ActivitySearchService",
		serviceVersion: 2,
	},
	AccommodationProductInfoRequest: {
		serviceName:    "AccommodationProductInfoService",
		servicePath:    "cmp.services.accommodation.v2.AccommodationProductInfoService",
		serviceVersion: 2,
	},
	AccommodationProductListRequest: {
		serviceName:    "AccommodationProductListService",
		servicePath:    "cmp.services.accommodation.v2.AccommodationProductListService",
		serviceVersion: 2,
	},
	AccommodationSearchRequest: {
		serviceName:    "AccommodationSearchService",
		servicePath:    "cmp.services.accommodation.v2.AccommodationSearchService",
		serviceVersion: 2,
	},
	MintRequest: {
		serviceName:    "MintService",
		servicePath:    "cmp.services.book.v2.MintService",
		serviceVersion: 2,
	},
	ValidationRequest: {
		serviceName:    "ValidationService",
		servicePath:    "cmp.services.book.v2.ValidationService",
		serviceVersion: 2,
	},
	TransportSearchRequest: {
		serviceName:    "TransportSearchService",
		servicePath:    "cmp.services.transport.v2.TransportSearchService",
		serviceVersion: 2,
	},
	SeatMapRequest: {
		serviceName:    "SeatMapService",
		servicePath:    "cmp.services.seat_map.v2.SeatMapService",
		serviceVersion: 2,
	},
	SeatMapAvailabilityRequest: {
		serviceName:    "SeatMapAvailabilityService",
		servicePath:    "cmp.services.seat_map.v2.SeatMapAvailabilityService",
		serviceVersion: 2,
	},
	CountryEntryRequirementsRequest: {
		serviceName:    "CountryEntryRequirementsService",
		servicePath:    "cmp.services.info.v2.CountryEntryRequirementsService",
		serviceVersion: 2,
	},
	InsuranceSearchRequest: {
		serviceName:    "InsuranceSearchService",
		servicePath:    "cmp.services.insurance.v1.InsuranceSearchService",
		serviceVersion: 1,
	},
	InsuranceProductInfoRequest: {
		serviceName:    "InsuranceProductInfoService",
		servicePath:    "cmp.services.insurance.v1.InsuranceProductInfoService",
		serviceVersion: 1,
	},
	InsuranceProductListRequest: {
		serviceName:    "InsuranceProductListService",
		servicePath:    "cmp.services.insurance.v1.InsuranceProductListService",
		serviceVersion: 1,
	},
	GetPartnerConfigurationRequest: {
		serviceName:    "GetPartnerConfigurationService",
		servicePath:    "cmp.services.partner.v2.GetPartnerConfigurationService",
		serviceVersion: 2,
	},
	GetNetworkFeeRequest: {
		serviceName:    "GetNetworkFeeService",
		servicePath:    "cmp.services.network.v1.GetNetworkFeeService",
		serviceVersion: 1,
	},
	PingRequest: {
		serviceName:    "PingService",
		servicePath:    "cmp.services.ping.v1.PingService",
		serviceVersion: 1,
	},
}
