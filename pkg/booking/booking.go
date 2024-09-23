package booking

import (
	"context"
	"crypto/ecdsa"
	"fmt"
	"math/big"
	"strings"

	"github.com/chain4travel/camino-messenger-contracts/go/contracts/cmaccount"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/ethclient"
	"go.uber.org/zap"
)

// Service provides minting and buying methods to interact with the CM Account contract.
type Service struct {
	client       *ethclient.Client
	logger       *zap.SugaredLogger
	cmAccount    *cmaccount.Cmaccount
	transactOpts *bind.TransactOpts
	chainID      *big.Int
}

// NewService initializes a new Service. It sets up the transactor with the provided
// private key and creates the CMAccount contract.
func NewService(
	cmAccountAddr common.Address,
	privateKey *ecdsa.PrivateKey,
	client *ethclient.Client,
	logger *zap.SugaredLogger,
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

	// TODO: If we need in future, we can set additional options like gas limit,
	// gas price, nonce, etc.
	//
	// transactOpts.GasLimit = uint64(300000) // example gas limit
	// transactOpts.GasPrice = big.NewInt(20000000000) // example gas price

	// Initialize the CMAccount
	cmAccount, err := cmaccount.NewCmaccount(cmAccountAddr, client)
	if err != nil {
		return nil, fmt.Errorf("failed to create CMAccount: %w", err)
	}

	return &Service{
		client:       client,
		logger:       logger,
		cmAccount:    cmAccount,
		transactOpts: transactOpts,
		chainID:      chainID,
	}, nil
}

// MintBookingToken mints a new booking token.
// Parameters:
// - reservedFor: The CM Account to reserve the token for.
// - uri: The URI of the token.
// - expirationTimestamp: Expiration timestamp for the token to be bought.
// - price: Price of the token.
// - paymentToken: Address of the payment token (ERC20), if address(0) then native.
func (bs *Service) MintBookingToken(
	reservedFor common.Address,
	uri string,
	expirationTimestamp *big.Int,
	price *big.Int,
	paymentToken common.Address,
) (*types.Transaction, error) {
	bs.logger.Infof("📅 Minting BookingToken for %s with price %s and expiration %s", reservedFor.Hex(), price, expirationTimestamp)

	// Validate URI
	// TODO: Should we have default tokenURI if no URI is provided?
	if strings.TrimSpace(uri) == "" {
		return nil, fmt.Errorf("uri cannot be empty")
	}

	// Call the MintBookingToken function from the contract
	tx, err := bs.cmAccount.MintBookingToken(
		bs.transactOpts,
		reservedFor,
		uri,
		expirationTimestamp,
		price,
		paymentToken,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to mint booking token: %w", err)
	}

	bs.logger.Infof("MintBookingToken tx sent: %s", tx.Hash().Hex())
	return tx, nil
}

// BuyBookingToken buys an existing reserved booking token.
// Parameters:
// - tokenId: ID of the token to buy.
func (bs *Service) BuyBookingToken(
	tokenID *big.Int,
) (*types.Transaction, error) {
	bs.logger.Infof("🛒 Buying BookingToken with TokenID %s", tokenID.String())

	// Validate tokenId
	if tokenID.Sign() <= 0 {
		return nil, fmt.Errorf("tokenId must be a positive integer")
	}

	// Call the BuyBookingToken function from the contract
	tx, err := bs.cmAccount.BuyBookingToken(bs.transactOpts, tokenID)
	if err != nil {
		return nil, fmt.Errorf("failed to buy booking token: %w", err)
	}

	bs.logger.Infof("BuyBookingToken tx sent: %s", tx.Hash().Hex())
	return tx, nil
}
