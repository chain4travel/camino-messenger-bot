/*
 * Copyright (C) 2022-2023, Chain4Travel AG. All rights reserved.
 * See the file LICENSE for licensing terms.
 */
package messaging

import (
	"context"
	"fmt"
	"sync"

	"buf.build/gen/go/chain4travel/camino-messenger-protocol/grpc/go/cmp/services/notification/v1/notificationv1grpc"
	"github.com/chain4travel/camino-messenger-bot/config"
	"github.com/chain4travel/camino-messenger-bot/internal/messaging/clients"
	"github.com/chain4travel/camino-messenger-bot/internal/messaging/types"
	"github.com/chain4travel/camino-messenger-bot/internal/rpc"
	"github.com/chain4travel/camino-messenger-bot/internal/rpc/client"
	"github.com/chain4travel/camino-messenger-contracts/go/contracts/cmaccount"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/ethclient"
	"go.uber.org/zap"
	grpc "google.golang.org/grpc"
	"google.golang.org/protobuf/reflect/protoreflect"
)

type ServiceRegistry interface {
	GetService(messageType types.MessageType) (Service, bool)

	// should only be called for supplier bot with rpc client
	NotificationClient() notificationv1grpc.NotificationServiceClient
}

func NewServiceRegistry(
	cfg *config.EvmConfig,
	evmClient *ethclient.Client,
	logger *zap.SugaredLogger,
	rpcClient *client.RPCClient,
) (ServiceRegistry, bool, error) {
	cmAccount, err := cmaccount.NewCmaccount(common.HexToAddress(cfg.CMAccountAddress), evmClient)
	if err != nil {
		return nil, false, fmt.Errorf("failed to fetch CM account: %w", err)
	}

	supportedServices, err := cmAccount.GetSupportedServices(&bind.CallOpts{})
	if err != nil {
		return nil, false, fmt.Errorf("failed to fetch Registered services: %w", err)
	}

	hasSupportedService := len(supportedServices.ServiceNames) > 0
	if hasSupportedService && rpcClient == nil {
		return nil, false, fmt.Errorf("bot supports some services, but doesn't have partner plugin rpc client configured")
	}

	services := make(map[types.MessageType]*service, len(supportedServices.ServiceNames))
	logStr := "\nSupported services:\n"
	for _, serviceName := range supportedServices.ServiceNames {
		logStr += serviceName + "\n"
		services[types.ServiceNameToRequestMessageType(serviceName)] = &service{name: serviceName}
	}
	logStr += "\n"
	logger.Info(logStr)

	registry := &serviceRegistry{
		logger:    logger,
		services:  services,
		lock:      &sync.RWMutex{},
		rpcClient: rpcClient,
	}

	registry.registerServices()

	return registry, hasSupportedService, nil
}

type serviceRegistry struct {
	logger    *zap.SugaredLogger
	services  map[types.MessageType]*service
	lock      *sync.RWMutex
	rpcClient *client.RPCClient
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

	rpc.Client
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
