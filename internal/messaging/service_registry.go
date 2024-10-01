/*
 * Copyright (C) 2022-2023, Chain4Travel AG. All rights reserved.
 * See the file LICENSE for licensing terms.
 */
package messaging

import (
	"fmt"
	"strings"
	"sync"

	"buf.build/gen/go/chain4travel/camino-messenger-protocol/grpc/go/cmp/services/notification/v1/notificationv1grpc"
	"github.com/chain4travel/camino-messenger-bot/internal/messaging/clients"
	"github.com/chain4travel/camino-messenger-bot/internal/messaging/messages"
	"github.com/chain4travel/camino-messenger-bot/internal/rpc/client"
	"github.com/chain4travel/camino-messenger-contracts/go/contracts/cmaccount"
	"go.uber.org/zap"
)

type ServiceRegistry interface {
	RegisterServices(rpcClient *client.RPCClient)
	GetClient(messageType messages.MessageType) (clients.Client, bool)

	// should only be called for supplier bot with rpc client
	NotificationClient() notificationv1grpc.NotificationServiceClient
}

type supportedServices struct {
	ServiceNames []string
	Services     []cmaccount.PartnerConfigurationService
}

type serviceRegistry struct {
	logger    *zap.SugaredLogger
	clients   map[messages.MessageType]clients.Client
	lock      *sync.RWMutex
	supported map[string]cmaccount.PartnerConfigurationService
	rpcClient *client.RPCClient
}

func NewServiceRegistry(supportedServices supportedServices, logger *zap.SugaredLogger) ServiceRegistry {
	supported := make(map[string]cmaccount.PartnerConfigurationService, len(supportedServices.ServiceNames))
	logStr := "\nSupported services:\n"
	for i, serviceFullName := range supportedServices.ServiceNames {
		logStr += serviceFullName + "\n"
		supported[serviceFullName] = supportedServices.Services[i]
	}
	logStr += "\n"
	logger.Info(logStr)

	return &serviceRegistry{
		logger:    logger,
		clients:   make(map[messages.MessageType]clients.Client),
		lock:      &sync.RWMutex{},
		supported: supported,
	}
}

func (s *serviceRegistry) RegisterServices(rpcClient *client.RPCClient) {
	s.lock.Lock()
	defer s.lock.Unlock()

	if s.isServiceSupported("cmp.services.ping.v1.PingService") {
		s.clients[messages.MessageType(s.getRequestTypeNameFromServiceName("PingService"))] =
			clients.NewPingServiceV1(rpcClient.ClientConn)
	}

	if rpcClient != nil {
		s.rpcClient = rpcClient
	}
}

func (s *serviceRegistry) getRequestTypeNameFromServiceName(name string) string {
	name = strings.TrimSuffix(name, "Service")
	return name + "Request"
}

func (s *serviceRegistry) GetClient(messageType messages.MessageType) (clients.Client, bool) {
	s.lock.RLock()
	defer s.lock.RUnlock()
	service, ok := s.clients[messageType]
	return service, ok
}

func (s *serviceRegistry) isServiceSupported(serviceFullName string) bool {
	_, ok := s.supported[serviceFullName]
	return ok
}

// func (s *serviceRegistry) GetClient(serviceFullName string) any {
// 	_, ok := s.supported[serviceFullName]
// 	if !ok {
// 		return nil
// 	}
// 	switch serviceFullName {
// 	case "cmp.services.activity.v2.ActivityProductInfoService":

// 	return notificationv1grpc.NewNotificationServiceClient(s.rpcClient.ClientConn)
// }

func (s *serviceRegistry) NotificationClient() notificationv1grpc.NotificationServiceClient {
	return notificationv1grpc.NewNotificationServiceClient(s.rpcClient.ClientConn)
}

func getServiceFullName(messageType messages.MessageType) (string, error) {
	serviceFullName, ok := servicesByMessageType[messageType]
	if !ok {
		return "", fmt.Errorf("service not found for message type %v", messageType)
	}
	return serviceFullName, nil
}

var servicesByMessageType = map[messages.MessageType]string{
	messages.ActivityProductInfoRequest:      "cmp.services.activity.v2.ActivityProductInfoService",
	messages.ActivityProductListRequest:      "cmp.services.activity.v2.ActivityProductListService",
	messages.ActivitySearchRequest:           "cmp.services.activity.v2.ActivitySearchService",
	messages.AccommodationProductInfoRequest: "cmp.services.accommodation.v2.AccommodationProductInfoService",
	messages.AccommodationProductListRequest: "cmp.services.accommodation.v2.AccommodationProductListService",
	messages.AccommodationSearchRequest:      "cmp.services.accommodation.v2.AccommodationSearchService",
	messages.MintRequest:                     "cmp.services.book.v2.MintService",
	messages.ValidationRequest:               "cmp.services.book.v2.ValidationService",
	messages.TransportSearchRequest:          "cmp.services.transport.v2.TransportSearchService",
	messages.SeatMapRequest:                  "cmp.services.seat_map.v2.SeatMapService",
	messages.SeatMapAvailabilityRequest:      "cmp.services.seat_map.v2.SeatMapAvailabilityService",
	messages.CountryEntryRequirementsRequest: "cmp.services.info.v2.CountryEntryRequirementsService",
	messages.InsuranceSearchRequest:          "cmp.services.insurance.v1.InsuranceSearchService",
	messages.InsuranceProductInfoRequest:     "cmp.services.insurance.v1.InsuranceProductInfoService",
	messages.InsuranceProductListRequest:     "cmp.services.insurance.v1.InsuranceProductListService",
	messages.GetPartnerConfigurationRequest:  "cmp.services.partner.v2.GetPartnerConfigurationService",
	messages.GetNetworkFeeRequest:            "cmp.services.network.v1.GetNetworkFeeService",
	messages.PingRequest:                     "cmp.services.ping.v1.PingService",
}
