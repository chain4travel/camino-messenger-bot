/*
 * Copyright (C) 2022-2023, Chain4Travel AG. All rights reserved.
 * See the file LICENSE for licensing terms.
 */

package messaging

import (
	"fmt"
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
	"github.com/chain4travel/camino-messenger-bot/config"
	"github.com/chain4travel/camino-messenger-bot/internal/rpc/client"
	"github.com/chain4travel/camino-messenger-contracts/go/contracts/cmaccount"
	"go.uber.org/zap"
)

type ServiceRegistry interface {
	RegisterServices(requestTypes config.SupportedRequestTypesFlag, rpcClient *client.RPCClient)
	GetService(messageType MessageType) (Service, bool)
}

type serviceRegistry struct {
	logger            *zap.SugaredLogger
	services          map[MessageType]Service
	supportedServices struct {
		ServiceNames []string
		Services     []cmaccount.PartnerConfigurationService
	}
	lock *sync.RWMutex
}
type Key struct {
	serviceName    string
	serviceVersion uint64
}

var supported map[Key]cmaccount.PartnerConfigurationService

func NewServiceRegistry(supportedServices struct {
	ServiceNames []string
	Services     []cmaccount.PartnerConfigurationService
}, logger *zap.SugaredLogger) ServiceRegistry {
	supported = make(map[Key]cmaccount.PartnerConfigurationService)

	//TODO: @VjeraTurk support multiple versions
	for i, serviceName := range supportedServices.ServiceNames {
		// Split each service name by "."
		servicePath := strings.Split(serviceName, ".")

		currentVersionString := servicePath[3]
		currentVersionString = currentVersionString[1:]
		num, err := strconv.ParseUint(currentVersionString, 10, 64)
		if err != nil {
			fmt.Println("Error:", err)
		}
		fmt.Println(servicePath[4], "registered version:", num)
		supported[Key{serviceName: servicePath[4], serviceVersion: num}] = supportedServices.Services[i]
	}

	return &serviceRegistry{
		logger:            logger,
		services:          make(map[MessageType]Service),
		supportedServices: supportedServices,
		lock:              &sync.RWMutex{},
	}
}

func (s *serviceRegistry) RegisterServices(requestTypes config.SupportedRequestTypesFlag, rpcClient *client.RPCClient) {
	s.lock.Lock()
	defer s.lock.Unlock()

	for _, requestType := range requestTypes {
		var service Service
		switch MessageType(requestType) {
		case ActivityProductInfoRequest:
			if isServiceVersionRegistered("ActivityProductInfoService", uint64(1)) {
				c := activityv1grpc.NewActivityProductInfoServiceClient(rpcClient.ClientConn)
				service = activityProductInfoService{client: &c}
			}
		case ActivityProductListRequest:
			if isServiceVersionRegistered("ActivityProductListService", uint64(1)) {
				c := activityv1grpc.NewActivityProductListServiceClient(rpcClient.ClientConn)
				service = activityProductListService{client: c}
			}
		case ActivitySearchRequest:
			if isServiceVersionRegistered("ActivitySearchService", uint64(1)) {
				c := activityv1grpc.NewActivitySearchServiceClient(rpcClient.ClientConn)
				service = activityService{client: &c}
			}
		case AccommodationProductInfoRequest:
			if isServiceVersionRegistered("AccommodationProductInfoService", uint64(1)) {
				c := accommodationv1grpc.NewAccommodationProductInfoServiceClient(rpcClient.ClientConn)
				service = accommodationProductInfoService{client: &c}
			}
		case AccommodationProductListRequest:
			if isServiceVersionRegistered("AccommodationProductListService", uint64(1)) {
				c := accommodationv1grpc.NewAccommodationProductListServiceClient(rpcClient.ClientConn)
				service = accommodationProductListService{client: &c}
			}
		case AccommodationSearchRequest:
			if isServiceVersionRegistered("AccommodationSearchService", uint64(1)) {
				c := accommodationv1grpc.NewAccommodationSearchServiceClient(rpcClient.ClientConn)
				service = accommodationSearchService{client: &c}
			}
		case MintRequest:
			if isServiceVersionRegistered("MintService", uint64(1)) {
				c := bookv1grpc.NewMintServiceClient(rpcClient.ClientConn)
				service = mintService{client: &c}
			}
			// WIP - support multiple version - otherwise linter complains
			if isServiceVersionRegistered("MintService", uint64(2)) {
				//c := bookv2grpc.NewMintServiceClient(rpcClient.ClientConn)
				//services[2] = mintService{client: &c}
				log.Print("supports MintService 2")
			}

		case ValidationRequest:
			if isServiceVersionRegistered("ValidationService", uint64(1)) {
				c := bookv1grpc.NewValidationServiceClient(rpcClient.ClientConn)
				service = validationService{client: &c}
			}
		case TransportSearchRequest:
			if isServiceVersionRegistered("TransportSearchService", uint64(1)) {
				c := transportv1grpc.NewTransportSearchServiceClient(rpcClient.ClientConn)
				service = transportService{client: &c}
			}
		case SeatMapRequest:
			if isServiceVersionRegistered("SeatMapService", uint64(1)) {
				c := seat_mapv1grpc.NewSeatMapServiceClient(rpcClient.ClientConn)
				service = seatMapService{client: &c}
			}
		case SeatMapAvailabilityRequest:
			if isServiceVersionRegistered("SeatMapAvailabilityService", uint64(1)) {
				c := seat_mapv1grpc.NewSeatMapAvailabilityServiceClient(rpcClient.ClientConn)
				service = seatMapAvailabilityService{client: &c}
			}
		case CountryEntryRequirementsRequest:
			if isServiceVersionRegistered("CountryEntryRequirementsService", uint64(1)) {
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

func isServiceVersionRegistered(name string, version uint64) bool {
	_, ok := supported[Key{serviceName: name, serviceVersion: version}]
	return ok
}
