package messaging

import (
	"context"
	"crypto/ecdsa"
	"fmt"
	"math/big"
	"time"

	"github.com/ava-labs/avalanchego/utils/crypto/secp256k1"
	"github.com/chain4travel/camino-messenger-bot/config"
	"github.com/chain4travel/camino-messenger-bot/internal/storage"
	"github.com/chain4travel/camino-messenger-bot/pkg/cheques"
	"github.com/chain4travel/camino-messenger-contracts/go/contracts/cmaccount"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/ethclient"
	"go.uber.org/zap"
)

var _ ChequeHandler = (*evmChequeHandler)(nil)

type evmChequeHandler struct {
	logger            *zap.SugaredLogger
	ethClient         *ethclient.Client
	evmConfig         *config.EvmConfig
	cmAccountAddress  common.Address
	cmAccountInstance *cmaccount.Cmaccount
	chainID           *big.Int
	privateKey        *ecdsa.PrivateKey

	cfg     *config.EvmConfig
	storage storage.Storage
}

type ChequeHandler interface {
	IssueCheque(ctx context.Context, fromCmAccount common.Address, toCmAccount common.Address, fromBot common.Address, toBot common.Address, amount uint64) (cheques.SignedCheque, error)
	getLastCashIn(ctx context.Context, fromCmAccount common.Address, toBot common.Address) (*LastCashIn, error)
	IsBotAllowed(ctx context.Context, fromBot common.Address) (bool, error)
	IsEmptyCheque(cheque cheques.SignedCheque) bool
}

type LastCashIn struct {
	Counter   *big.Int
	Amount    *big.Int
	CreatedAt *big.Int
	ExpiresAt *big.Int
}

func NewChequeHandler(logger *zap.SugaredLogger, ethClient *ethclient.Client, evmConfig *config.EvmConfig, chainID *big.Int, storage storage.Storage) (ChequeHandler, error) {
	pk := new(secp256k1.PrivateKey)

	if err := pk.UnmarshalText([]byte("\"" + evmConfig.PrivateKey + "\"")); err != nil {
		logger.Fatalf("Failed to parse private key: %v", err)
	}
	ecdsaPk := pk.ToECDSA()

	cmAccountAddress := common.HexToAddress(evmConfig.CMAccountAddress)
	cmAccountInstance, err := cmaccount.NewCmaccount(cmAccountAddress, ethClient)
	if err != nil {
		return nil, fmt.Errorf("failed to instantiate contract binding: %v", err)
	}

	return &evmChequeHandler{
		ethClient:         ethClient,
		cmAccountAddress:  cmAccountAddress,
		cmAccountInstance: cmAccountInstance,
		chainID:           chainID,
		privateKey:        ecdsaPk,
		logger:            logger,
		evmConfig:         evmConfig,
		storage:           storage,
	}, nil
}

func (cm *evmChequeHandler) IssueCheque(ctx context.Context, fromCMAccount common.Address, toCMAccount common.Address, fromBot common.Address, toBot common.Address, amount uint64) (cheques.SignedCheque, error) {
	session, err := cm.newStorageSession(ctx)
	if err != nil {
		return cheques.SignedCheque{}, fmt.Errorf("failed to create session: %w", err)
	}

	previousChequeModel, err := session.GetLatestChequeRecordFromBotPair(ctx, fromBot, toBot)
	if err != nil {
		return cheques.SignedCheque{}, fmt.Errorf("failed to get previous cheque: %w", err)
	}

	counter := previousChequeModel.Counter.Add(previousChequeModel.Counter, big.NewInt(1))

	lastCashIn, err := cm.cmAccountInstance.GetLastCashIn(&bind.CallOpts{}, fromBot, toBot)
	if err != nil {
		return cheques.SignedCheque{}, fmt.Errorf("failed to get last cash in: %w", err)
	}
	if lastCashIn.LastCounter.Cmp(counter) > 0 {
		return cheques.SignedCheque{}, fmt.Errorf("invalid cheque counter: counter is less than last cash in counter from sc")
	}

	bigIntAmount := new(big.Int)
	bigIntAmount.SetUint64(amount)

	totalAmount, err := cm.GetTotalAmountFromBotPair(ctx, fromBot, toBot)
	totalAmount = totalAmount.Add(totalAmount, bigIntAmount)

	newCheque := &cheques.Cheque{
		FromCMAccount: fromCMAccount,
		ToCMAccount:   toCMAccount,
		ToBot:         toBot,
		Counter:       counter,
		Amount:        totalAmount,
		CreatedAt:     big.NewInt(time.Now().Unix()),
		ExpiresAt:     big.NewInt(0),
	}
	signer, err := cheques.NewSigner(cm.privateKey, cm.chainID)
	signedCheque, err := signer.SignCheque(newCheque)

	previousCheque := &cheques.Cheque{
		FromCMAccount: previousChequeModel.FromCMAccount,
		ToCMAccount:   previousChequeModel.ToCMAccount,
		ToBot:         previousChequeModel.ToBot,
		Counter:       previousChequeModel.Counter,
		Amount:        previousChequeModel.Amount,
		CreatedAt:     previousChequeModel.CreatedAt,
		ExpiresAt:     previousChequeModel.ExpiresAt,
	}

	previousChequeSigned, err := signer.SignCheque(previousCheque)

	verified := cheques.VerifyCheque(previousChequeSigned, signedCheque, big.NewInt(time.Now().Unix()), big.NewInt(0))
	if verified != nil {
		return cheques.SignedCheque{}, fmt.Errorf("failed to verify cheque: %v", verified)
	}

	return *signedCheque, nil
}

func (cm *evmChequeHandler) getLastCashIn(ctx context.Context, fromBot common.Address, toBot common.Address) (*LastCashIn, error) {
	// Use contract binding to interact with the contract
	callOpts := &bind.CallOpts{
		Context: ctx,
	}

	cashIn, err := cm.cmAccountInstance.GetLastCashIn(callOpts, fromBot, toBot)
	if err != nil {
		return nil, fmt.Errorf("failed to get last cash in: %w", err)
	}

	// Return as LastCashIn struct
	return &LastCashIn{
		Counter:   cashIn.LastCounter,
		Amount:    cashIn.LastAmount,
		CreatedAt: cashIn.LastCreatedAt,
		ExpiresAt: cashIn.LastExpiresAt,
	}, nil
}

func (cm *evmChequeHandler) buildTransactOpts(ctx context.Context) (*bind.TransactOpts, error) {
	auth, err := bind.NewKeyedTransactorWithChainID(cm.privateKey, cm.chainID)
	if err != nil {
		return nil, fmt.Errorf("failed to create keyed transactor: %v", err)
	}
	auth.Context = ctx
	auth.GasLimit = uint64(300000) // You can adjust the gas limit accordingly

	return auth, nil
}

func (cm *evmChequeHandler) GetTotalAmountFromBotPair(ctx context.Context, fromBot common.Address, toBot common.Address) (*big.Int, error) {
	session, err := cm.newStorageSession(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to create new storage session: %w", err)
	}
	cheques, err := session.GetChequesByBotPair(ctx, fromBot, toBot)
	if err != nil {
		return nil, fmt.Errorf("failed to get cheques by bot pair: %w", err)
	}

	totalAmount := big.NewInt(0)

	for _, cheque := range cheques {
		totalAmount.Add(totalAmount, cheque.Amount)
	}

	return totalAmount, nil
}

func (cm *evmChequeHandler) newStorageSession(ctx context.Context) (storage.Session, error) {
	session, err := cm.storage.NewSession(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to create new storage session: %w", err)
	}

	if commitErr := session.Commit(); commitErr != nil {
		return nil, fmt.Errorf("failed to commit session: %w", commitErr)
	}

	return session, nil
}

func (cm *evmChequeHandler) IsEmptyCheque(cheque cheques.SignedCheque) bool {
	return cheque.Cheque.FromCMAccount == common.Address{} && cheque.Cheque.ToCMAccount == common.Address{} && cheque.Cheque.ToBot == common.Address{} && cheque.Cheque.Counter == nil && cheque.Cheque.Amount == nil && cheque.Cheque.CreatedAt == nil && cheque.Cheque.ExpiresAt == nil
}

func (cm *evmChequeHandler) IsBotAllowed(ctx context.Context, fromBot common.Address) (bool, error) {
	callOpts := &bind.CallOpts{
		Context: ctx,
	}
	isAllowed, err := cm.cmAccountInstance.IsBotAllowed(callOpts, fromBot)
	if err != nil {
		return false, fmt.Errorf("bot does not have necessary permission: %v", err)
	}

	return isAllowed, nil
}
