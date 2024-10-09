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
	"github.com/metachris/eth-go-bindings/erc20"
	"go.uber.org/zap"
)

// Service provides minting and buying methods to interact with the CM Account contract.
type Service struct {
	client           *ethclient.Client
	logger           *zap.SugaredLogger
	cmAccount        *cmaccount.Cmaccount
	transactOpts     *bind.TransactOpts
	chainID          *big.Int
	cmAccountAddress *common.Address
}

var zeroAddress = common.HexToAddress("0x0000000000000000000000000000000000000000")

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
		client:           client,
		logger:           logger,
		cmAccount:        cmAccount,
		transactOpts:     transactOpts,
		chainID:          chainID,
		cmAccountAddress: &cmAccountAddr,
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
	if paymentToken != zeroAddress {
		erc20Contract, err := erc20.NewErc20(paymentToken, bs.client)
		if err != nil {
			return nil, fmt.Errorf("failed to instantiate ERC20 contract: %w", err)
		}

		if err := bs.checkAndApproveAllowance(context.Background(), erc20Contract, reservedFor, *bs.cmAccountAddress, price); err != nil {
			return nil, fmt.Errorf("error during token approval process: %w", err)
		}
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

// convertPriceToBigInt converts the price to its integer representation
func (bs *Service) ConvertPriceToBigInt(value string, decimals int32, totalDecimals int32) (*big.Int, error) {
	// Convert the value string to a big.Int
	valueBigInt := new(big.Int)
	_, ok := valueBigInt.SetString(value, 10)
	if !ok {
		return nil, fmt.Errorf("failed to convert value to big.Int: %s", value)
	}

	// Calculate the multiplier as 10^(totalDecimals - price.Decimals)
	multiplier := new(big.Int).Exp(big.NewInt(10), big.NewInt(int64(totalDecimals-decimals)), nil)

	// Multiply the value by the multiplier
	result := new(big.Int).Mul(valueBigInt, multiplier)

	return result, nil
}

// checkAndApproveAllowance checks if the allowance is sufficient and approves tokens if necessary
func (bs *Service) checkAndApproveAllowance(
	ctx context.Context,
	erc20Contract *erc20.Erc20,
	owner, spender common.Address,
	price *big.Int,
) error {
	// Check allowance
	allowance, err := erc20Contract.Allowance(&bind.CallOpts{Context: ctx}, owner, spender)
	if err != nil {
		return fmt.Errorf("failed to get allowance: %w", err)
	}
	bs.logger.Infof("current allowance: %s", allowance.String())

	// If allowance is less than the price, approve more tokens
	if allowance.Cmp(price) < 0 {
		bs.logger.Infof("Allowance insufficient. Initiating approval for the required amount...")

		// Approve the required amount
		approveTx, err := erc20Contract.Approve(bs.transactOpts, spender, price)
		if err != nil {
			return fmt.Errorf("failed to approve token spending: %s", err)
		}

		bs.logger.Infof("Approval transaction sent: %s", approveTx.Hash().Hex())

		// Wait for the approval transaction to be mined
		receipt, err := bind.WaitMined(ctx, bs.client, approveTx)
		if err != nil {
			return fmt.Errorf("failed to wait for approval transaction to be mined: %w", err)
		}

		if receipt.Status != types.ReceiptStatusSuccessful {
			return fmt.Errorf("approval transaction failed: %v", receipt)
		}

		bs.logger.Info("Approval transaction mined successfully.")
	} else {
		bs.logger.Infof("Sufficient allowance available. Proceeding with the transaction...")
	}

	return nil
}
