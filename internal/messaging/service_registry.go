/*
 * Copyright (C) 2022-2023, Chain4Travel AG. All rights reserved.
 * See the file LICENSE for licensing terms.
 */
package messaging

import (
	"strings"
	"sync"

	"buf.build/gen/go/chain4travel/camino-messenger-protocol/grpc/go/cmp/services/notification/v1/notificationv1grpc"
	"github.com/chain4travel/camino-messenger-bot/internal/messaging/clients"
	"github.com/chain4travel/camino-messenger-bot/internal/messaging/types"
	"github.com/chain4travel/camino-messenger-bot/internal/rpc/client"
	"github.com/chain4travel/camino-messenger-contracts/go/contracts/cmaccount"
	"go.uber.org/zap"
)

type ServiceRegistry interface {
	RegisterServices(rpcClient *client.RPCClient)
	GetClient(messageType types.MessageType) (clients.Client, bool)

	// should only be called for supplier bot with rpc client
	NotificationClient() notificationv1grpc.NotificationServiceClient
}

type supportedServices struct {
	ServiceNames []string
	Services     []cmaccount.PartnerConfigurationService
}

type serviceRegistry struct {
	logger    *zap.SugaredLogger
	clients   map[types.MessageType]clients.Client
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
		clients:   make(map[types.MessageType]clients.Client),
		lock:      &sync.RWMutex{},
		supported: supported,
	}
}

func (s *serviceRegistry) RegisterServices(rpcClient *client.RPCClient) {
	s.lock.Lock()
	defer s.lock.Unlock()

	if s.isServiceSupported("cmp.services.ping.v1.PingService") {
		// TODO@ will not work, because we clients is MessageType -> Client
		// TODO@ and we need to have different service versions for same MessageType
		s.clients[s.getRequestTypeNameFromServiceName("PingService")] = clients.NewPingServiceV1(rpcClient.ClientConn)
	}

	if rpcClient != nil {
		s.rpcClient = rpcClient
	}
}

func (s *serviceRegistry) getRequestTypeNameFromServiceName(name string) types.MessageType {
	return types.MessageType(strings.TrimSuffix(name, "Service") + "Request")
}

func (s *serviceRegistry) GetClient(messageType types.MessageType) (clients.Client, bool) {
	s.lock.RLock()
	defer s.lock.RUnlock()
	service, ok := s.clients[messageType]
	return service, ok
}

func (s *serviceRegistry) isServiceSupported(serviceFullName string) bool {
	_, ok := s.supported[serviceFullName]
	return ok
}

func (s *serviceRegistry) NotificationClient() notificationv1grpc.NotificationServiceClient {
	return notificationv1grpc.NewNotificationServiceClient(s.rpcClient.ClientConn)
}
