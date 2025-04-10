// Copyright (C) 2022-2025, Chain4Travel AG. All rights reserved.
// See the file LICENSE for licensing terms.

package booking

import (
	"context"
	"crypto/ecdsa"
	"fmt"
	"math/big"
	"strings"

	cmaccounts "github.com/chain4travel/camino-messenger-bot/pkg/cm_accounts"
	"github.com/chain4travel/camino-messenger-contracts/go/contracts/bookingtoken"

	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/ethclient"
	"go.uber.org/zap"
)

var (
	_ Service = (*service)(nil)

	// Special address that indicates BookingToken payment will be in native coin of
	// the network (CAM).
	NativePaymentToken = common.HexToAddress("0x0000000000000000000000000000000000000000")

	// Special address that indicates BookingToken payment will occur off-chain.
	ISOPaymentToken = common.HexToAddress("0x0000000000000000000000000000000000000001")
)

type Status uint8

const (
	StatusUnspecified Status = iota
	StatusReserved
	StatusReservationExpired
	StatusBought
	StatusCancelled
)

// Service provides minting and buying methods to interact with the CM Account contract.
type Service interface {
	// MintBookingToken mints a new booking token.
	// Parameters:
	// - reservedFor: The CM Account to reserve the token for.
	// - uri: The URI of the token.
	// - expirationTimestamp: Expiration timestamp for the token to be bought.
	// - price: Price of the token.
	// - paymentToken: Address of the payment token (ERC20), if address(0) then native.
	// - offChainPaymentCurrency: The currency to be used for off-chain payments.
	// Returns the transaction receipt.
	MintBookingToken(
		ctx context.Context,
		reservedFor common.Address,
		uri string,
		expirationTimestamp *big.Int,
		price *big.Int,
		paymentToken common.Address,
		offChainPaymentCurrency *big.Int,
	) (*types.Receipt, *big.Int, error)

	// BuyBookingToken buys an existing reserved booking token.
	// Parameters:
	// - tokenId: ID of the token to buy.
	// - price: Price of the token.
	// - paymentToken: Address of the payment token (ERC20), if address(0) then native.
	// Returns the transaction receipt.
	BuyBookingToken(
		ctx context.Context,
		tokenID *big.Int,
		price *big.Int,
		paymentToken common.Address,
	) (*types.Receipt, error)
}

type service struct {
	logger                 *zap.SugaredLogger
	transactOpts           *bind.TransactOpts
	chainID                *big.Int
	minterCMAccountAddress common.Address
	cmAccounts             cmaccounts.Service
	bookingToken           *bookingtoken.Bookingtoken
}

// NewService initializes a new Service. It sets up the transactor with the provided
// private key and creates the CMAccount contract.
func NewService(
	ethClient *ethclient.Client,
	bookingTokenAddress common.Address,
	minterCMAccountAddress common.Address,
	privateKey *ecdsa.PrivateKey,
	chainID *big.Int,
	logger *zap.SugaredLogger,
	cmAccounts cmaccounts.Service,
) (Service, error) {
	bookingToken, err := bookingtoken.NewBookingtoken(bookingTokenAddress, ethClient)
	if err != nil {
		logger.Errorf("failed to create booking token contract binding: %v", err)
		return nil, err
	}

	transactOpts, err := bind.NewKeyedTransactorWithChainID(privateKey, chainID)
	if err != nil {
		return nil, fmt.Errorf("failed to create transactor: %w", err)
	}

	// TODO @havan If we need in future, we can set additional options like gas limit, gas price, nonce, etc.
	// transactOpts.GasLimit = uint64(300000) // example gas limit
	// transactOpts.GasPrice = big.NewInt(20000000000) // example gas price

	return &service{
		logger:                 logger,
		transactOpts:           transactOpts,
		chainID:                chainID,
		minterCMAccountAddress: minterCMAccountAddress,
		cmAccounts:             cmAccounts,
		bookingToken:           bookingToken,
	}, nil
}

func (bs *service) MintBookingToken(
	ctx context.Context,
	reservedFor common.Address,
	uri string,
	expirationTimestamp *big.Int,
	price *big.Int,
	paymentToken common.Address,
	offChainPaymentCurrency *big.Int,
) (*types.Receipt, *big.Int, error) {
	bs.logger.Infof("📅 Minting BookingToken for %s with price %s and expiration %s", reservedFor.Hex(), price, expirationTimestamp)

	// Validate URI
	// TODO: Should we have default tokenURI if no URI is provided?
	if strings.TrimSpace(uri) == "" {
		return nil, nil, fmt.Errorf("uri cannot be empty")
	}
	// Call the MintBookingToken function from the contract

	receipt, err := bs.cmAccounts.MintBookingToken(
		ctx,
		bs.transactOpts,
		bs.minterCMAccountAddress,
		reservedFor,
		uri,
		expirationTimestamp,
		price,
		paymentToken,
		offChainPaymentCurrency,
	)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to mint booking token: %w", err)
	}

	for _, log := range receipt.Logs {
		if event, err := bs.bookingToken.ParseTokenReserved(*log); err == nil {
			bs.logger.Infof("[TokenReserved] TokenID: %s ReservedFor: %s Price: %s, PaymentToken: %s", event.TokenId, event.ReservedFor, event.Price, event.PaymentToken)
			return receipt, event.TokenId, nil
		}
	}

	return receipt, nil, fmt.Errorf("failed to parse TokenReserved event from tx receipt logs (txID: %s)", receipt.TxHash.Hex())
}

func (bs *service) BuyBookingToken(
	ctx context.Context,
	tokenID *big.Int,
	price *big.Int,
	paymentToken common.Address,
) (*types.Receipt, error) {
	bs.logger.Infof("🛒 Buying BookingToken with TokenID %s", tokenID.String())

	// Validate tokenId
	if tokenID.Sign() < 0 {
		return nil, fmt.Errorf("tokenId must be a positive integer (>= 0)")
	}

	// Call the BuyBookingToken function from the contract
	receipt, err := bs.cmAccounts.BuyBookingToken(ctx, bs.transactOpts, bs.minterCMAccountAddress, tokenID, price, paymentToken)
	if err != nil {
		return nil, fmt.Errorf("failed to buy booking token: %w", err)
	}

	bs.logger.Infof("BuyBookingToken tx sent: %s", receipt.TxHash.Hex())
	return receipt, nil
}
