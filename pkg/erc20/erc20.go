package erc20

import (
	"context"
	"fmt"
	"math/big"

	"github.com/chain4travel/camino-messenger-contracts/go/contracts/erc20"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/ethclient"
	lru "github.com/hashicorp/golang-lru/v2"
)

var _ Service = (*erc20Service)(nil)

type Service interface {
	Decimals(ctx context.Context, contractAddress common.Address) (int32, error)

	Allowance(
		ctx context.Context,
		contractAddress common.Address,
		owner common.Address,
		spender common.Address,
	) (*big.Int, error)

	Approve(
		ctx context.Context,
		transactOpts *bind.TransactOpts,
		contractAddress common.Address,
		spender common.Address,
		price *big.Int,
	) error
}

type erc20Service struct {
	erc20Cache *lru.Cache[common.Address, *erc20.Erc20]
	ethClient  *ethclient.Client
}

func NewERC20Service(ethClient *ethclient.Client, erc20CacheSize int) (Service, error) {
	erc20Cache, err := lru.New[common.Address, *erc20.Erc20](erc20CacheSize)
	if err != nil {
		return nil, err
	}

	return &erc20Service{
		erc20Cache: erc20Cache,
		ethClient:  ethClient,
	}, nil
}

func (s *erc20Service) erc20(contractAddress common.Address) (*erc20.Erc20, error) {
	tokenContract, found := s.erc20Cache.Get(contractAddress)
	if !found {
		var err error
		tokenContract, err = erc20.NewErc20(contractAddress, s.ethClient)
		if err != nil {
			return nil, fmt.Errorf("failed to instantiate ERC20 contract: %w", err)
		}
		s.erc20Cache.Add(contractAddress, tokenContract)
	}

	return tokenContract, nil
}

func (s *erc20Service) Decimals(ctx context.Context, contractAddress common.Address) (int32, error) {
	tokenContract, err := s.erc20(contractAddress)
	if err != nil {
		return 0, err
	}

	decimals, err := tokenContract.Decimals(&bind.CallOpts{Context: ctx})
	if err != nil {
		return 0, fmt.Errorf("failed to fetch token decimals: %w", err)
	}

	return int32(decimals), nil
}

func (s *erc20Service) Allowance(
	ctx context.Context,
	contractAddress common.Address,
	owner common.Address,
	spender common.Address,
) (*big.Int, error) {
	tokenContract, err := s.erc20(contractAddress)
	if err != nil {
		return nil, err
	}

	allowance, err := tokenContract.Allowance(&bind.CallOpts{Context: ctx}, owner, spender)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch allowance: %w", err)
	}

	return allowance, nil
}

func (s *erc20Service) Approve(
	ctx context.Context,
	transactOpts *bind.TransactOpts,
	contractAddress common.Address,
	spender common.Address,
	price *big.Int,
) error {
	tokenContract, err := s.erc20(contractAddress)
	if err != nil {
		return err
	}

	approveTx, err := tokenContract.Approve(transactOpts, spender, price)
	if err != nil {
		return fmt.Errorf("failed to approve token spending: %w", err)
	}

	// Wait for the approval transaction to be mined
	receipt, err := bind.WaitMined(ctx, s.ethClient, approveTx)
	if err != nil {
		return fmt.Errorf("failed to wait for approval transaction to be mined: %w", err)
	}

	if receipt.Status != types.ReceiptStatusSuccessful {
		return fmt.Errorf("approval transaction failed: %v", receipt)
	}

	return nil
}
