// Copyright (C) 2022-2026, Chain4Travel AG. All rights reserved.
// See the file LICENSE for licensing terms.

package booking

import (
	"context"
	"crypto/ecdsa"
	"fmt"
	"math/big"
	"strings"

	cmaccounts "github.com/chain4travel/camino-messenger-bot/v13/pkg/cm_accounts"
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

	errEmptyURI = fmt.Errorf("uri cannot be empty")
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
	// - isCancellable: Indicates if the booking can be cancelled.
	// Returns the transaction receipt.
	MintBookingToken(
		ctx context.Context,
		reservedFor common.Address,
		uri string,
		expirationTimestamp *big.Int,
		price *big.Int,
		paymentToken common.Address,
		offChainPaymentCurrency *big.Int,
		isCancellable bool,
	) (*types.Receipt, *big.Int, error)

	// BuyBookingToken buys an existing reserved booking token.
	// Parameters:
	// - tokenID: ID of the token to buy.
	// - price: Price of the token.
	// - paymentToken: Address of the payment token (ERC20), if address(0) then native.
	// Returns the transaction receipt.
	BuyBookingToken(
		ctx context.Context,
		tokenID *big.Int,
		price *big.Int,
		paymentToken common.Address,
	) (*types.Receipt, error)

	// RecordExpiration records the expiration of a booking token on-chain.
	// Parameters:
	// - tokenID: ID of the token to record expiration for.
	// Returns the transaction receipt.
	RecordExpiration(
		ctx context.Context,
		tokenID *big.Int,
	) (*types.Receipt, error)

	// GetBookingStatus retrieves the booking status of a token.
	// Parameters:
	// - blockNumber: Block number to query the status at. If nil, the latest block is used.
	// - tokenID: ID of the token to get the status for.
	// Returns the booking status.
	GetBookingStatus(
		ctx context.Context,
		blockNumber *big.Int,
		tokenID *big.Int,
	) (Status, error)

	// IsBookingCancellable checks if a booking is cancellable.
	// Parameters:
	// - blockNumber: Block number to query the status at. If nil, the latest block is used.
	// - tokenID: ID of the token to get the status for.
	// Returns the booking status.
	IsBookingCancellable(
		ctx context.Context,
		blockNumber *big.Int,
		tokenID *big.Int,
	) (bool, error)

	// GetCancellationReasonsByTx retrieves cancellation reasons event from given tx.
	// Parameters:
	// - txHash: Hash of the BookingtokenCancellationReasons event transaction.
	GetCancellationReasonsEvent(
		ctx context.Context,
		txHash common.Hash,
	) (*bookingtoken.BookingtokenCancellationReasons, error)

	// GetCancellationReasons retrieves cancellation reasons from token cancellation proposal.
	// Parameters:
	// - blockNumber: Block number to query the reasons at. If nil, the latest block is used.
	// - tokenID: ID of the token to get the cancellation reasons from.
	GetCancellationReasons(
		ctx context.Context,
		blockNumber *big.Int,
		tokenID *big.Int,
	) (*CancellationReasons, error)

	// GetCancellationProposal retrieves cancellation proposal from token.
	// Parameters:
	// - blockNumber: Block number to query the proposal at. If nil, the latest block is used.
	// - tokenID: ID of the token to get the cancellation proposal from.
	GetCancellationProposal(
		ctx context.Context,
		blockNumber *big.Int,
		tokenID *big.Int,
	) (*CancellationProposal, error)
}

type service struct {
	logger                 *zap.SugaredLogger
	transactOpts           *bind.TransactOpts
	chainID                *big.Int
	minterCMAccountAddress common.Address
	cmAccounts             cmaccounts.Service
	bookingToken           *bookingtoken.Bookingtoken
	ethClient              *ethclient.Client
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
		return nil, fmt.Errorf("failed to create booking token contract binding: %w", err)
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
		ethClient:              ethClient,
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
	isCancellable bool,
) (*types.Receipt, *big.Int, error) {
	bs.logger.Infof("📅 Minting BookingToken for %s with price %s and expiration %s", reservedFor.Hex(), price, expirationTimestamp)

	// Validate URI
	// TODO: Should we have default tokenURI if no URI is provided?
	if strings.TrimSpace(uri) == "" {
		return nil, nil, errEmptyURI
	}

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
		isCancellable,
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

	// Validate tokenID
	if tokenID.Sign() < 0 {
		return nil, fmt.Errorf("tokenID must be a positive integer (>= 0)")
	}

	// Call the BuyBookingToken function from the contract
	receipt, err := bs.cmAccounts.BuyBookingToken(ctx, bs.transactOpts, bs.minterCMAccountAddress, tokenID, price, paymentToken)
	if err != nil {
		return nil, fmt.Errorf("failed to buy booking token: %w", err)
	}

	return receipt, nil
}

func (bs *service) RecordExpiration(
	ctx context.Context,
	tokenID *big.Int,
) (*types.Receipt, error) {
	bs.logger.Infof("📝 Recording expiration for BookingToken with TokenID %s", tokenID.String())

	// Validate tokenID
	if tokenID.Sign() < 0 {
		return nil, fmt.Errorf("tokenID must be a positive integer (>= 0)")
	}

	receipt, err := bs.cmAccounts.RecordExpiration(ctx, bs.transactOpts, bs.minterCMAccountAddress, tokenID)
	if err != nil {
		return nil, fmt.Errorf("failed to record token expiration: %w", err)
	}

	return receipt, nil
}

func (bs *service) GetBookingStatus(
	ctx context.Context,
	blockNumber *big.Int,
	tokenID *big.Int,
) (Status, error) {
	status, err := bs.bookingToken.GetBookingStatus(&bind.CallOpts{BlockNumber: blockNumber, Context: ctx}, tokenID)
	if err != nil {
		return StatusUnspecified, fmt.Errorf("failed to get booking status: %w", err)
	}
	return Status(status), nil
}

func (bs *service) IsBookingCancellable(
	ctx context.Context,
	blockNumber *big.Int,
	tokenID *big.Int,
) (bool, error) {
	isCancellable, err := bs.bookingToken.IsCancellable(&bind.CallOpts{BlockNumber: blockNumber, Context: ctx}, tokenID)
	if err != nil {
		return false, fmt.Errorf("failed to get booking cancellable status: %w", err)
	}
	return isCancellable, nil
}

func (bs *service) GetCancellationReasonsEvent(
	ctx context.Context,
	txHash common.Hash,
) (*bookingtoken.BookingtokenCancellationReasons, error) {
	txReceipt, err := bs.ethClient.TransactionReceipt(ctx, txHash)
	switch {
	case err != nil:
		return nil, fmt.Errorf("failed to get transaction receipt: %w", err)
	case txReceipt == nil || txReceipt.Status != types.ReceiptStatusSuccessful:
		return nil, fmt.Errorf("transaction receipt not found or failed for txHash: %s", txHash.Hex())
	}

	for _, log := range txReceipt.Logs {
		if event, err := bs.bookingToken.ParseCancellationReasons(*log); err == nil {
			return event, nil
		}
	}

	return nil, fmt.Errorf("failed to parse CancellationReasons event from tx receipt logs (txID: %s)", txHash.Hex())
}

type CancellationReasons struct {
	CancellationReason  uint16
	CancellationVersion uint16
	RejectionReason     uint16
	RejectionVersion    uint16
	CounterReason       uint16
	CounterVersion      uint16
	WithdrawalReason    uint16
	WithdrawalVersion   uint16
}

func (bs *service) GetCancellationReasons(
	ctx context.Context,
	blockNumber *big.Int,
	tokenID *big.Int,
) (*CancellationReasons, error) {
	reasons, err := bs.bookingToken.GetCancellationReasons(&bind.CallOpts{BlockNumber: blockNumber, Context: ctx}, tokenID)
	if err != nil {
		return nil, fmt.Errorf("failed to get cancellation reasons: %w", err)
	}
	return (*CancellationReasons)(&reasons), nil
}

type CancellationProposalStatus uint8

const (
	CancellationProposalStatusNoProposal CancellationProposalStatus = iota
	CancellationProposalStatusPending
	CancellationProposalStatusRejected
	CancellationProposalStatusWithdrawn
	CancellationProposalStatusFinalized
)

type CancellationProposal struct {
	Status           CancellationProposalStatus
	RefundAmount     *big.Int
	InitialProposer  common.Address
	CurrentProposer  common.Address
	OwnerAccepted    bool
	SupplierAccepted bool
	TimesCountered   uint32
	TimesRejected    uint32
}

func (bs *service) GetCancellationProposal(
	ctx context.Context,
	blockNumber *big.Int,
	tokenID *big.Int,
) (*CancellationProposal, error) {
	status, refundAmount, initialProposer, currentProposer, ownerAccepted, supplierAccepted, timesCountered, timesRejected, err := bs.bookingToken.GetCancellationProposal(
		&bind.CallOpts{BlockNumber: blockNumber, Context: ctx},
		tokenID,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to get cancellation proposal: %w", err)
	}
	return &CancellationProposal{
		Status:           CancellationProposalStatus(status),
		RefundAmount:     refundAmount,
		InitialProposer:  initialProposer,
		CurrentProposer:  currentProposer,
		OwnerAccepted:    ownerAccepted,
		SupplierAccepted: supplierAccepted,
		TimesCountered:   timesCountered,
		TimesRejected:    timesRejected,
	}, nil
}
