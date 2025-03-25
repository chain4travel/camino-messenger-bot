// Copyright (C) 2022-2025, Chain4Travel AG. All rights reserved.
// See the file LICENSE for licensing terms.

package cmaccounts

import (
	"context"
	"crypto/ecdsa"
	"errors"
	"fmt"
	"math/big"

	"github.com/chain4travel/camino-messenger-bot/pkg/cheques"
	"github.com/chain4travel/camino-messenger-contracts/go/contracts/cmaccount"
	"github.com/chain4travel/camino-messenger-contracts/go/contracts/cmaccountmanager"

	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethclient"
	lru "github.com/hashicorp/golang-lru/v2"
	"go.uber.org/zap"
)

const (
	// Implementation slot for ERC1967Proxy
	// See: https://eips.ethereum.org/EIPS/eip-1967#logic-contract-address
	managerCMAccountImplementationSlotString = "0x360894a13ba1a3210667c828492db98dca3e2076cc3735a920a3ca505d382bbc"
)

var (
	_ Service = &service{}

	bigZero                            = big.NewInt(0)
	chequeOperatorRole                 = crypto.Keccak256Hash([]byte("CHEQUE_OPERATOR_ROLE"))
	managerCMAccountImplementationSlot = common.HexToHash(managerCMAccountImplementationSlotString)

	ErrorNoChequeOperators        = errors.New("no cheque operators found (no bots found in cmAccount)")
	ErrorUnableToObtainServiceFee = errors.New("unable to obtain service fee")
)

type Service interface {
	GetFirstChequeOperator(ctx context.Context, cmAccountAddress common.Address) (common.Address, error)

	VerifyCheque(ctx context.Context, cheque *cheques.SignedCheque) (bool, error)

	CashInCheque(
		ctx context.Context,
		cheque *cheques.SignedCheque,
		botKey *ecdsa.PrivateKey,
	) (common.Hash, error)

	GetServiceFee(
		ctx context.Context,
		cmAccountAddress common.Address,
		serviceFullName string,
	) (*big.Int, error)

	IsBotAllowed(
		ctx context.Context,
		cmAccountAddress common.Address,
		botAddress common.Address,
	) (bool, error)

	GetLastCashIn(
		ctx context.Context,
		cmAccountAddress common.Address,
		fromBot common.Address,
		toBot common.Address,
	) (counter *big.Int, amount *big.Int, err error)

	MintBookingToken(
		ctx context.Context,
		transactOpts *bind.TransactOpts,
		cmAccountAddress common.Address,
		reservedFor common.Address,
		uri string,
		expirationTimestamp *big.Int,
		price *big.Int,
		paymentToken common.Address,
		isoCurrency *big.Int,
	) (*types.Receipt, error)

	BuyBookingToken(
		ctx context.Context,
		transactOpts *bind.TransactOpts,
		cmAccountAddr common.Address,
		tokenID *big.Int,
		price *big.Int,
		paymentToken common.Address,
	) (*types.Receipt, error)

	IsCMAccountImplementationUpToDate(ctx context.Context, cmAccountAddress common.Address) (bool, error)
}

type service struct {
	ethClient *ethclient.Client
	cache     *lru.Cache[common.Address, *cmaccount.Cmaccount]
	logger    *zap.SugaredLogger
	chainID   *big.Int
}

func NewService(
	logger *zap.SugaredLogger,
	cacheSize int,
	ethClient *ethclient.Client,
) (Service, error) {
	chainID, err := ethClient.ChainID(context.Background())
	if err != nil {
		logger.Errorf("Failed to get chain ID: %v", err)
		return nil, err
	}

	cache, err := lru.New[common.Address, *cmaccount.Cmaccount](cacheSize)
	if err != nil {
		return nil, err
	}

	return &service{
		ethClient: ethClient,
		cache:     cache,
		logger:    logger,
		chainID:   chainID,
	}, nil
}

func (s *service) GetFirstChequeOperator(ctx context.Context, cmAccountAddress common.Address) (common.Address, error) {
	cmAccount, err := s.cmAccount(cmAccountAddress)
	if err != nil {
		s.logger.Errorf("Failed to get cm account: %v", err)
		return common.Address{}, err
	}

	countBig, err := cmAccount.GetRoleMemberCount(&bind.CallOpts{Context: ctx}, chequeOperatorRole)
	if err != nil {
		s.logger.Errorf("Failed to get role member count: %v", err)
		return common.Address{}, err
	}

	if countBig.Cmp(bigZero) <= 0 { // count <= 0
		s.logger.Error(ErrorNoChequeOperators)
		return common.Address{}, ErrorNoChequeOperators
	}

	botsAddress, err := cmAccount.GetRoleMember(&bind.CallOpts{Context: ctx}, chequeOperatorRole, big.NewInt(0))
	if err != nil {
		s.logger.Errorf("Failed to get role member: %v", err)
		return common.Address{}, err
	}
	return botsAddress, nil
}

func (s *service) CashInCheque(
	ctx context.Context,
	cheque *cheques.SignedCheque,
	botKey *ecdsa.PrivateKey,
) (common.Hash, error) {
	cmAccount, err := s.cmAccount(cheque.FromCMAccount)
	if err != nil {
		s.logger.Errorf("failed to get cmAccount contract instance: %v", err)
		return common.Hash{}, err
	}

	transactor, err := bind.NewKeyedTransactorWithChainID(botKey, s.chainID)
	if err != nil {
		s.logger.Error(err)
		return common.Hash{}, err
	}
	transactor.Context = ctx

	tx, err := cmAccount.CashInCheque(
		transactor,
		cheque.FromCMAccount,
		cheque.ToCMAccount,
		cheque.ToBot,
		cheque.Counter,
		cheque.Amount,
		cheque.CreatedAt,
		cheque.ExpiresAt,
		cheque.Signature,
	)
	if err != nil {
		s.logger.Errorf("failed to cash in cheque %s: %v", cheque, err)
		return common.Hash{}, err
	}

	return tx.Hash(), nil
}

func (s *service) VerifyCheque(ctx context.Context, cheque *cheques.SignedCheque) (bool, error) {
	cmAccount, err := s.cmAccount(cheque.FromCMAccount)
	if err != nil {
		s.logger.Errorf("failed to get cmAccount contract instance: %v", err)
		return false, err
	}

	_, err = cmAccount.VerifyCheque(
		&bind.CallOpts{Context: ctx},
		cheque.FromCMAccount,
		cheque.ToCMAccount,
		cheque.ToBot,
		cheque.Counter,
		cheque.Amount,
		cheque.CreatedAt,
		cheque.ExpiresAt,
		cheque.Signature,
	)
	if err != nil && err.Error() == "execution reverted" {
		return false, nil
	}
	return err == nil, err
}

func (s *service) GetServiceFee(
	ctx context.Context,
	cmAccountAddress common.Address,
	serviceFullName string,
) (*big.Int, error) {
	cmAccount, err := s.cmAccount(cmAccountAddress)
	if err != nil {
		return nil, fmt.Errorf("failed to get supplier cmAccount: %w", err)
	}

	serviceFee, err := cmAccount.GetServiceFee(
		&bind.CallOpts{Context: ctx},
		serviceFullName,
	)
	if err != nil {
		return nil, fmt.Errorf("%w: %w", ErrorUnableToObtainServiceFee, err)
	}
	return serviceFee, nil
}

func (s *service) IsBotAllowed(
	ctx context.Context,
	cmAccountAddress common.Address,
	botAddress common.Address,
) (bool, error) {
	cmAccount, err := s.cmAccount(cmAccountAddress)
	if err != nil {
		return false, fmt.Errorf("failed to get cmAccount contract instance: %w", err)
	}

	isBotAllowed, err := cmAccount.IsBotAllowed(
		&bind.CallOpts{Context: ctx},
		botAddress,
	)
	if err != nil {
		return false, fmt.Errorf("failed to check if bot is allowed: %w", err)
	}
	return isBotAllowed, nil
}

func (s *service) GetLastCashIn(
	ctx context.Context,
	cmAccountAddress common.Address,
	fromBot common.Address,
	toBot common.Address,
) (counter *big.Int, amount *big.Int, err error) {
	cmAccount, err := s.cmAccount(cmAccountAddress)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to get cmAccount contract instance: %w", err)
	}

	lastCashIn, err := cmAccount.GetLastCashIn(
		&bind.CallOpts{Context: ctx},
		fromBot,
		toBot,
	)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to get last cash in: %w", err)
	}
	return lastCashIn.LastCounter, lastCashIn.LastAmount, nil
}

func (s *service) MintBookingToken(
	ctx context.Context,
	transactOpts *bind.TransactOpts,
	cmAccountAddress common.Address,
	reservedFor common.Address,
	uri string,
	expirationTimestamp *big.Int,
	price *big.Int,
	paymentToken common.Address,
	offchainPaymentCurrency *big.Int,
) (*types.Receipt, error) {
	cmAccount, err := s.cmAccount(cmAccountAddress)
	if err != nil {
		return nil, fmt.Errorf("failed to get cmAccount contract instance: %w", err)
	}

	// TODO: @VjeraTurk enable setting isCancellable flag from mintv3 on.
	tx, err := cmAccount.MintBookingToken(
		transactOpts,
		reservedFor,
		uri,
		expirationTimestamp,
		price,
		paymentToken,
		offchainPaymentCurrency,
		false, // In mintv1 and mintv2 isCancellable is always false - as tokens are not cancellable.
	)
	if err != nil {
		return nil, fmt.Errorf("failed to mint booking token: %w", err)
	}

	receipt, err := bind.WaitMined(ctx, s.ethClient, tx)
	if err != nil {
		return nil, err
	}

	if receipt.Status != types.ReceiptStatusSuccessful {
		return nil, fmt.Errorf("transaction failed: %v", receipt)
	}

	return receipt, nil
}

func (s *service) BuyBookingToken(
	ctx context.Context,
	transactOpts *bind.TransactOpts,
	cmAccountAddress common.Address,
	tokenID *big.Int,
	price *big.Int,
	paymentToken common.Address,
) (*types.Receipt, error) {
	cmAccount, err := s.cmAccount(cmAccountAddress)
	if err != nil {
		return nil, fmt.Errorf("failed to get cmAccount contract instance: %w", err)
	}

	tx, err := cmAccount.BuyBookingToken(
		transactOpts,
		tokenID,
		price,
		paymentToken,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to buy booking token: %w", err)
	}

	s.logger.Infof("Waiting for BuyBookingToken transaction to be mined...\n")

	receipt, err := bind.WaitMined(ctx, s.ethClient, tx)
	if err != nil {
		return nil, err
	}

	if receipt.Status != types.ReceiptStatusSuccessful {
		return nil, fmt.Errorf("transaction failed: %v", receipt)
	}

	s.logger.Infof("Successfully mined. Block Nr: %s Gas used: %d\n", receipt.BlockNumber, receipt.GasUsed)

	return receipt, nil
}

func (s *service) cmAccount(cmAccountAddr common.Address) (*cmaccount.Cmaccount, error) {
	cmAccount, ok := s.cache.Get(cmAccountAddr)
	if ok {
		return cmAccount, nil
	}

	cmaccount, err := cmaccount.NewCmaccount(cmAccountAddr, s.ethClient)
	if err != nil {
		return nil, err
	}
	s.cache.Add(cmAccountAddr, cmaccount)

	return cmaccount, nil
}

func (s *service) getLatestCMAccountImplementation(ctx context.Context, cmAccountAddress common.Address) (common.Address, error) {
	cmAccount, err := s.cmAccount(cmAccountAddress)
	if err != nil {
		return common.Address{}, fmt.Errorf("failed to fetch CM account: %w", err)
	}
	managerAddress, err := cmAccount.GetManagerAddress(&bind.CallOpts{Context: ctx})
	if err != nil {
		return common.Address{}, fmt.Errorf("failed to fetch CM account Manager Address: %w", err)
	}
	manager, err := cmaccountmanager.NewCmaccountmanager(managerAddress, s.ethClient)
	if err != nil {
		return common.Address{}, fmt.Errorf("failed to get Manager: %w", err)
	}
	currentImplOnManager, err := manager.GetAccountImplementation(&bind.CallOpts{Context: ctx})
	if err != nil {
		return common.Address{}, fmt.Errorf("failed to get Account Implementation: %w", err)
	}
	return currentImplOnManager, nil
}

func (s *service) getCurrentImplementationOnProxy(ctx context.Context, cmAccountAddress common.Address) (common.Address, error) {
	implAddressSlotValue, err := s.ethClient.StorageAt(ctx, cmAccountAddress, managerCMAccountImplementationSlot, nil)
	if err != nil {
		return common.Address{}, fmt.Errorf("failed to get implementation address from proxy: %w", err)
	}
	return slotValueToAddress(implAddressSlotValue)
}

func (s *service) IsCMAccountImplementationUpToDate(ctx context.Context, cmAccountAddress common.Address) (bool, error) {
	currentImplOnManager, err := s.getLatestCMAccountImplementation(ctx, cmAccountAddress)
	if err != nil {
		return false, fmt.Errorf("failed to get current implementation on manager: %w", err)
	}

	currentImplOnProxy, err := s.getCurrentImplementationOnProxy(ctx, cmAccountAddress)
	if err != nil {
		return false, fmt.Errorf("failed to get current implementation on proxy: %w", err)
	}
	s.logger.Info("📜 CM Account Implementation:")
	s.logger.Info("   - Active:  " + currentImplOnProxy.Hex())
	s.logger.Info("   - Latest:  " + currentImplOnManager.Hex())

	return currentImplOnManager == currentImplOnProxy, nil
}

func slotValueToAddress(slotValue []byte) (common.Address, error) {
	if len(slotValue) < 32 {
		return common.Address{}, fmt.Errorf("slot value storage read returned unexpected size: %d", len(slotValue))
	}
	// We take the last 20 bytes because Ethereum addresses are 20 bytes (40 hex chars) but storage slots are 32 bytes
	// The address is right-aligned in the 32 byte slot
	return common.BytesToAddress(slotValue[12:]), nil
}
