package cancellation

import (
	"context"
	"crypto/ecdsa"
	"fmt"
	"math/big"

	cmaccounts "github.com/chain4travel/camino-messenger-bot/pkg/cm_accounts"
	"github.com/chain4travel/camino-messenger-bot/pkg/erc20"

	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/ethclient"
	"go.uber.org/zap"
)

// Service provides minting and buying methods to interact with the CM Account contract.
type Service struct {
	client           *ethclient.Client
	logger           *zap.SugaredLogger
	transactOpts     *bind.TransactOpts
	chainID          *big.Int
	cmAccountAddress common.Address
	erc20            erc20.Service
	cmAccounts       cmaccounts.Service
}

// NewService initializes a new Service. It sets up the transactor with the provided
// private key and creates the CMAccount contract.
func NewService(
	cmAccountAddr common.Address,
	privateKey *ecdsa.PrivateKey,
	client *ethclient.Client,
	logger *zap.SugaredLogger,
	erc20 erc20.Service,
	cmAccounts cmaccounts.Service,
) (*Service, error) {
	// Get the chain ID to prevent replay attacks
	chainID, err := client.ChainID(context.Background())
	if err != nil {
		return nil, fmt.Errorf("failed to get network ID: %w", err)
	}

	// Create TransactOpts
	transactOpts, err := bind.NewKeyedTransactorWithChainID(privateKey, chainID)
	if err != nil {
		return nil, fmt.Errorf("failed to create transactor: %w", err)
	}

	return &Service{
		client:           client,
		logger:           logger,
		transactOpts:     transactOpts,
		chainID:          chainID,
		cmAccountAddress: cmAccountAddr,
		erc20:            erc20,
		cmAccounts:       cmAccounts,
	}, nil
}

func (cs *Service) InitiateCancellationProposal(
	ctx context.Context,
	tokenID *big.Int,
	refoundAmount *big.Int,
) (*types.Receipt, error) {
	receipt, err := cs.cmAccounts.InitiateCancellationProposal(
		ctx,
		cs.transactOpts,
		cs.cmAccountAddress,
		tokenID,
		refoundAmount,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to initiate Cancel Proposal: %w", err)
	}

	return receipt, nil
}

func (cs *Service) SetCancellable(
	ctx context.Context,
	tokenID *big.Int,
	isCancelable bool,
) (*types.Receipt, error) {
	receipt, err := cs.cmAccounts.SetCancellable(
		ctx,
		cs.transactOpts,
		cs.cmAccountAddress, // OR?
		tokenID,
		isCancelable,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to set Token Cancellable: %w", err)
	}

	return receipt, nil
}
