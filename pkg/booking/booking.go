package booking

import (
	"context"
	"crypto/ecdsa"
	"fmt"
	"math/big"
	"strings"

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

var zeroAddress = common.HexToAddress("0x0000000000000000000000000000000000000000")

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

	// TODO: If we need in future, we can set additional options like gas limit,
	// gas price, nonce, etc.
	//
	// transactOpts.GasLimit = uint64(300000) // example gas limit
	// transactOpts.GasPrice = big.NewInt(20000000000) // example gas price

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

// MintBookingToken mints a new booking token.
// Parameters:
// - reservedFor: The CM Account to reserve the token for.
// - uri: The URI of the token.
// - expirationTimestamp: Expiration timestamp for the token to be bought.
// - price: Price of the token.
// - paymentToken: Address of the payment token (ERC20), if address(0) then native.
func (bs *Service) MintBookingToken(
	ctx context.Context,
	reservedFor common.Address,
	uri string,
	expirationTimestamp *big.Int,
	price *big.Int,
	paymentToken common.Address,
) (*types.Receipt, error) {
	bs.logger.Infof("📅 Minting BookingToken for %s with price %s and expiration %s", reservedFor.Hex(), price, expirationTimestamp)

	// Validate URI
	// TODO: Should we have default tokenURI if no URI is provided?
	if strings.TrimSpace(uri) == "" {
		return nil, fmt.Errorf("uri cannot be empty")
	}
	// if paymentToken is zeroAddress, then it is either native token or iso currency payment
	if paymentToken != zeroAddress {
		if err := bs.checkAndApproveAllowance(ctx, paymentToken, reservedFor, bs.cmAccountAddress, price); err != nil {
			return nil, fmt.Errorf("error during token approval process: %w", err)
		}
	}
	// Call the MintBookingToken function from the contract

	receipt, err := bs.cmAccounts.MintBookingToken(
		ctx,
		bs.transactOpts,
		bs.cmAccountAddress,
		reservedFor,
		uri,
		expirationTimestamp,
		price,
		paymentToken,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to mint booking token: %w", err)
	}

	bs.logger.Infof("MintBookingToken tx sent: %s", receipt.TxHash.Hex())
	return receipt, nil
}

// BuyBookingToken buys an existing reserved booking token.
// Parameters:
// - tokenId: ID of the token to buy.
func (bs *Service) BuyBookingToken(
	ctx context.Context,
	tokenID *big.Int,
) (*types.Receipt, error) {
	bs.logger.Infof("🛒 Buying BookingToken with TokenID %s", tokenID.String())

	// Validate tokenId
	if tokenID.Sign() <= 0 {
		return nil, fmt.Errorf("tokenId must be a positive integer")
	}

	// Call the BuyBookingToken function from the contract
	receipt, err := bs.cmAccounts.BuyBookingToken(ctx, bs.transactOpts, tokenID)
	if err != nil {
		return nil, fmt.Errorf("failed to buy booking token: %w", err)
	}

	bs.logger.Infof("BuyBookingToken tx sent: %s", receipt.TxHash.Hex())
	return receipt, nil
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
	paymentToken common.Address,
	owner, spender common.Address,
	price *big.Int,
) error {
	// Check allowance
	allowance, err := bs.erc20.Allowance(ctx, paymentToken, owner, spender)
	if err != nil {
		return fmt.Errorf("failed to get allowance: %w", err)
	}
	bs.logger.Infof("current allowance: %s", allowance.String())

	// If allowance is less than the price, approve more tokens
	if allowance.Cmp(price) < 0 {
		bs.logger.Infof("Allowance insufficient. Initiating approval for the required amount...")

		// Approve the required amount

		if err := bs.erc20.Approve(ctx, bs.transactOpts, paymentToken, spender, price); err != nil {
			return fmt.Errorf("failed to approve token spending: %w", err)
		}

		bs.logger.Info("Approval transaction mined successfully.")
	} else {
		bs.logger.Infof("Sufficient allowance available. Proceeding with the transaction...")
	}

	return nil
}
