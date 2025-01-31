// Copyright (C) 2022-2025, Chain4Travel AG. All rights reserved.
// See the file LICENSE for licensing terms.

package cmaccounts

import (
	"context"
	"fmt"

	"github.com/chain4travel/camino-messenger-contracts/go/contracts/cmaccount"
	"github.com/chain4travel/camino-messenger-contracts/go/contracts/cmaccountmanager"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/ethclient"
	"go.uber.org/zap"
)

type CmAccountService interface {
	WarnIfUpgradeNeeded() error
}

type cmAccountService struct {
	cmAccountAddress *common.Address
	cmAccount        *cmaccount.Cmaccount
	manager          *cmaccountmanager.Cmaccountmanager
	logger           *zap.SugaredLogger
	ethClient        *ethclient.Client
}

func NewCmAccountService(
	cmAccountAddress common.Address,
	logger *zap.SugaredLogger,
	ethClient *ethclient.Client,
) (CmAccountService, error) {
	cmAccount, err := cmaccount.NewCmaccount(cmAccountAddress, ethClient)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch CM account: %w", err)
	}

	managerAddress, err := cmAccount.GetManagerAddress(&bind.CallOpts{})
	if err != nil {
		return nil, fmt.Errorf("failed to fetch CM account Manager Address: %w", err)
	}
	manager, err := cmaccountmanager.NewCmaccountmanager(managerAddress, ethClient)
	if err != nil {
		return nil, fmt.Errorf("failed to get Manager: %w", err)
	}

	return &cmAccountService{
		cmAccountAddress: &cmAccountAddress,
		cmAccount:        cmAccount,
		manager:          manager,
		logger:           logger,
		ethClient:        ethClient,
	}, nil
}

func (s *cmAccountService) WarnIfUpgradeNeeded() error {
	currentImplOnManager, err := s.manager.GetAccountImplementation(&bind.CallOpts{})
	if err != nil {
		return fmt.Errorf("failed to get Account Implementation: %w", err)
	}

	// Implementation slot for ERC1967Proxy
	implementationSlot := common.HexToHash("0x360894a13ba1a3210667c828492db98dca3e2076cc3735a920a3ca505d382bbc")

	// Read implementation from proxy
	implAddress, err := s.ethClient.StorageAt(context.Background(), *s.cmAccountAddress, implementationSlot, nil)
	if err != nil {
		return fmt.Errorf("failed to get implementation address from proxy: %w", err)
	}

	// Convert to address (last 20 bytes)
	currentImplOnProxy := common.BytesToAddress(implAddress[12:])

	s.logger.Info("Implementation:")
	s.logger.Info("   - Active:  " + currentImplOnProxy.Hex())
	s.logger.Info("   - Latest:  " + currentImplOnManager.Hex())

	if currentImplOnProxy != currentImplOnManager {
		// TODO: @VjeraTurk Ensure multiple versions compatibility
		// TODO: @VjeraTurk Inform about the consequences of not upgrading for specific cases (bookingtoken, manager, cmaccount ...)
		s.logger.Error("CMAccount needs an upgrade!")
	} else {
		s.logger.Info("CMAccount is using the latest implementation.")
	}
	return nil
}
