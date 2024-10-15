/*
 * Copyright (C) 2022-2023, Chain4Travel AG. All rights reserved.
 * See the file LICENSE for licensing terms.
 */
package messaging

import (
	"errors"
	"fmt"

	"buf.build/gen/go/chain4travel/camino-messenger-protocol/grpc/go/cmp/services/notification/v1/notificationv1grpc"
	"github.com/chain4travel/camino-messenger-bot/internal/messaging/types"
	"github.com/chain4travel/camino-messenger-bot/internal/rpc"
	"github.com/chain4travel/camino-messenger-bot/internal/rpc/client"
	"github.com/chain4travel/camino-messenger-bot/internal/rpc/generated"
	"github.com/chain4travel/camino-messenger-contracts/go/contracts/cmaccount"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/ethclient"
	"go.uber.org/zap"
)

var errUnsupportedService = errors.New("cm account support service, which bot doesn't support")

type ServiceRegistry interface {
	GetService(messageType types.MessageType) (rpc.Service, bool)

	// should only be called for supplier bot with rpc client
	NotificationClient() notificationv1grpc.NotificationServiceClient
}

func NewServiceRegistry(
	cmAccountAddress common.Address,
	evmClient *ethclient.Client,
	logger *zap.SugaredLogger,
	rpcClient *client.RPCClient,
) (ServiceRegistry, error) {
	cmAccount, err := cmaccount.NewCmaccount(cmAccountAddress, evmClient)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch CM account: %w", err)
	}

	supportedServices, err := cmAccount.GetSupportedServices(&bind.CallOpts{})
	if err != nil {
		return nil, fmt.Errorf("failed to fetch Registered services: %w", err)
	}

	hasSupportedServices := len(supportedServices.ServiceNames) > 0
	partnerPluginClientEnabled := rpcClient != nil
	var services map[types.MessageType]rpc.Service

	switch {
	case hasSupportedServices && !partnerPluginClientEnabled:
		return nil, fmt.Errorf("bot supports some services, but doesn't have partner plugin rpc client enabled")
	case !hasSupportedServices && partnerPluginClientEnabled:
		logger.Warn("Bot doesn't support any services, but has partner plugin rpc client enabled")
	case hasSupportedServices && partnerPluginClientEnabled: // register supported services
		servicesNames := make(map[string]struct{}, len(supportedServices.ServiceNames))

		logStr := "\nSupported services:\n"
		for _, serviceName := range supportedServices.ServiceNames {
			logStr += serviceName + "\n"
			servicesNames[serviceName] = struct{}{}
		}

		services = generated.RegisterClientServices(rpcClient.ClientConn, servicesNames)

		logStr += "\n"
		logger.Info(logStr)

		if len(servicesNames) > 0 {
			logger.Error(errUnsupportedService)

			logStr := "\nUnsupported services:\n"
			for serviceName := range servicesNames {
				logStr += serviceName + "\n"
			}
			logStr += "\n"
			logger.Warn(logStr)

			return nil, errUnsupportedService
		}
	}

	return &serviceRegistry{
		logger:    logger,
		services:  services,
		rpcClient: rpcClient,
	}, nil
}

type serviceRegistry struct {
	logger    *zap.SugaredLogger
	services  map[types.MessageType]rpc.Service
	rpcClient *client.RPCClient
}

func (s *serviceRegistry) GetService(requestType types.MessageType) (rpc.Service, bool) {
	service, ok := s.services[requestType]
	if !ok {
		return nil, false
	}
	return service, true
}

func (s *serviceRegistry) NotificationClient() notificationv1grpc.NotificationServiceClient {
	return notificationv1grpc.NewNotificationServiceClient(s.rpcClient.ClientConn)
}
