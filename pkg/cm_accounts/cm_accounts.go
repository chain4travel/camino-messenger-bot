package cmaccounts

import (
	"context"
	"crypto/ecdsa"
	"fmt"
	"math/big"

	"github.com/chain4travel/camino-messenger-bot/pkg/cheques"
	"github.com/chain4travel/camino-messenger-contracts/go/contracts/cmaccount"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethclient"
	lru "github.com/hashicorp/golang-lru/v2"
	"go.uber.org/zap"
)

var (
	_ Service = &service{}

	chequeOperatorRole = crypto.Keccak256Hash([]byte("CHEQUE_OPERATOR_ROLE"))
)

type Service interface {
	GetChequeOperators(ctx context.Context, cmAccountAddress common.Address) ([]common.Address, error)

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
	) (*types.Receipt, error)

	BuyBookingToken(
		ctx context.Context,
		transactOpts *bind.TransactOpts,
		tokenID *big.Int,
	) (*types.Receipt, error)
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

type service struct {
	ethClient *ethclient.Client
	cache     *lru.Cache[common.Address, *cmaccount.Cmaccount]
	logger    *zap.SugaredLogger
	chainID   *big.Int
}

func (s *service) GetChequeOperators(ctx context.Context, cmAccountAddress common.Address) ([]common.Address, error) {
	cmAccount, err := s.cmAccount(cmAccountAddress)
	if err != nil {
		s.logger.Errorf("Failed to get cm account: %v", err)
		return nil, err
	}

	countBig, err := cmAccount.GetRoleMemberCount(&bind.CallOpts{Context: ctx}, chequeOperatorRole)
	if err != nil {
		s.logger.Errorf("Failed to call contract function: %v", err)
		return nil, err
	}

	count := countBig.Int64()
	botsAddresses := make([]common.Address, 0, count)
	for i := int64(0); i < count; i++ {
		address, err := cmAccount.GetRoleMember(&bind.CallOpts{Context: ctx}, chequeOperatorRole, big.NewInt(i))
		if err != nil {
			s.logger.Errorf("Failed to call contract function: %v", err)
			continue
		}
		botsAddresses = append(botsAddresses, address)
	}

	return botsAddresses, nil
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
		return nil, fmt.Errorf("failed to get service fee: %w", err)
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
) (*types.Receipt, error) {
	cmAccount, err := s.cmAccount(cmAccountAddress)
	if err != nil {
		return nil, fmt.Errorf("failed to get cmAccount contract instance: %w", err)
	}

	tx, err := cmAccount.MintBookingToken(
		transactOpts,
		reservedFor,
		uri,
		expirationTimestamp,
		price,
		paymentToken,
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
	tokenID *big.Int,
) (*types.Receipt, error) {
	cmAccount, err := s.cmAccount(transactOpts.From)
	if err != nil {
		return nil, fmt.Errorf("failed to get cmAccount contract instance: %w", err)
	}

	tx, err := cmAccount.BuyBookingToken(
		transactOpts,
		tokenID,
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
