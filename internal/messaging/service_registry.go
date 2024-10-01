/*
 * Copyright (C) 2022-2023, Chain4Travel AG. All rights reserved.
 * See the file LICENSE for licensing terms.
 */
package messaging

import (
	"context"
	"sync"

	"buf.build/gen/go/chain4travel/camino-messenger-protocol/grpc/go/cmp/services/notification/v1/notificationv1grpc"
	"github.com/chain4travel/camino-messenger-bot/internal/messaging/clients"
	"github.com/chain4travel/camino-messenger-bot/internal/messaging/types"
	"github.com/chain4travel/camino-messenger-bot/internal/rpc/client"
	"github.com/chain4travel/camino-messenger-contracts/go/contracts/cmaccount"
	"go.uber.org/zap"
	grpc "google.golang.org/grpc"
	"google.golang.org/protobuf/reflect/protoreflect"
)

type ServiceRegistry interface {
	RegisterServices(rpcClient *client.RPCClient)
	GetService(messageType types.MessageType) (Service, bool)

	// should only be called for supplier bot with rpc client
	NotificationClient() notificationv1grpc.NotificationServiceClient
}

type SupportedServices struct {
	ServiceNames []string
	Services     []cmaccount.PartnerConfigurationService
}

type serviceRegistry struct {
	logger    *zap.SugaredLogger
	services  map[types.MessageType]*service
	lock      *sync.RWMutex
	rpcClient *client.RPCClient
}

func NewServiceRegistry(supportedServices SupportedServices, logger *zap.SugaredLogger) ServiceRegistry {
	services := make(map[types.MessageType]*service, len(supportedServices.ServiceNames))
	logStr := "\nSupported services:\n"
	for _, serviceName := range supportedServices.ServiceNames {
		logStr += serviceName + "\n"
		services[types.ServiceNameToRequestMessageType(serviceName)] = &service{name: serviceName}
	}
	logStr += "\n"
	logger.Info(logStr)

	return &serviceRegistry{
		logger:   logger,
		services: services,
		lock:     &sync.RWMutex{},
	}
}

func (s *serviceRegistry) RegisterServices(rpcClient *client.RPCClient) {
	s.lock.Lock()
	defer s.lock.Unlock()

	if srv, ok := s.services[clients.PingServiceV1Request]; ok {
		srv.client = clients.NewPingServiceV1(rpcClient.ClientConn)
	}

	if rpcClient != nil {
		s.rpcClient = rpcClient
	}
}

func (s *serviceRegistry) GetService(requestType types.MessageType) (Service, bool) {
	s.lock.RLock()
	defer s.lock.RUnlock()
	service, ok := s.services[requestType]
	if !ok {
		return nil, false
	}
	return service, true
}

func (s *serviceRegistry) NotificationClient() notificationv1grpc.NotificationServiceClient {
	return notificationv1grpc.NewNotificationServiceClient(s.rpcClient.ClientConn)
}

var _ Service = (*service)(nil)

type Service interface {
	Name() string

	clients.Client
}

type service struct {
	client clients.Client
	name   string
}

func (s *service) Call(ctx context.Context, request protoreflect.ProtoMessage, opts ...grpc.CallOption) (protoreflect.ProtoMessage, types.MessageType, error) {
	return s.client.Call(ctx, request, opts...)
}

func (s *service) Name() string {
	return s.name
}
