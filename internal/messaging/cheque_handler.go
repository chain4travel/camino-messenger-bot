package messaging

import (
	"context"
	"crypto/ecdsa"
	"fmt"
	"math/big"
	"time"

	"github.com/ava-labs/avalanchego/utils/crypto/secp256k1"
	"github.com/chain4travel/camino-messenger-bot/config"

	// "github.com/chain4travel/camino-messenger-bot/internal/models"
	"github.com/chain4travel/camino-messenger-bot/internal/storage"
	"github.com/chain4travel/camino-messenger-bot/pkg/cheques"
	"github.com/chain4travel/camino-messenger-contracts/go/contracts/cmaccount"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
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
	// chequeRecordID is a hash of fromBot, toBot and toCmAccount
	chequeRecordID := chequeRecordID(fromBot, toBot, toCMAccount)
	previousChequeModel, err := session.GetLatestChequeRecord(ctx, chequeRecordID)
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

	bigIntAmount := new(big.Int).SetUint64(amount)
	totalAmount := new(big.Int).Add(previousChequeModel.Amount, bigIntAmount)

	if totalAmount.Cmp(lastCashIn.LastAmount) < 0 {
		// Return an error indicating that the amount is invalid
		return cheques.SignedCheque{}, fmt.Errorf("total amount (%s) is smaller than last cash-in amount (%s)", totalAmount.String(), lastCashIn.LastAmount.String())
	}

	newCheque := &cheques.Cheque{
		FromCMAccount: fromCMAccount,
		ToCMAccount:   toCMAccount,
		ToBot:         toBot,
		Counter:       counter,
		Amount:        bigIntAmount,
		CreatedAt:     big.NewInt(time.Now().Unix()),
		ExpiresAt:     big.NewInt(0),
	}
	signer, err := cheques.NewSigner(cm.privateKey, cm.chainID)
	if err != nil {
		return cheques.SignedCheque{}, fmt.Errorf("failed to create signer: %v", err)
	}
	signedCheque, err := signer.SignCheque(newCheque)
	if err != nil {
		return cheques.SignedCheque{}, fmt.Errorf("failed to sign cheque: %v", err)
	}
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
	if err != nil {
		return cheques.SignedCheque{}, fmt.Errorf("failed to sign previous cheque: %v", err)
	}

	verified := cheques.VerifyCheque(previousChequeSigned, signedCheque, big.NewInt(time.Now().Unix()), big.NewInt(0))
	if verified != nil {
		return cheques.SignedCheque{}, fmt.Errorf("failed to verify cheque: %v", verified)
	}

	sc, err := cm.verifyWithSmartContract(ctx, *signedCheque)
	if err != nil {
		return cheques.SignedCheque{}, fmt.Errorf("failed to verify cheque with smart contract: %v", err)
	}

	// insert into db
	// UpsertChequeRecord(ctx, models.ChequeRecord{
	// 	ChequeRecordID: chequeRecordID,
	// 	TxID:           nil,
	// 	Status:         ChequeTxStatusPending,
	// 	signedCheque:   signedCheque,
	// })

	return *signedCheque, nil
}

func (cm *evmChequeHandler) verifyWithSmartContract(ctx context.Context, signedCheque cheques.SignedCheque) (struct {
	Signer        common.Address
	PaymentAmount *big.Int
}, error) {
	result, err := cm.cmAccountInstance.VerifyCheque(
		&bind.CallOpts{Context: ctx},
		signedCheque.Cheque.FromCMAccount,
		signedCheque.Cheque.ToCMAccount,
		signedCheque.Cheque.ToBot,
		signedCheque.Cheque.Counter,
		signedCheque.Cheque.Amount,
		signedCheque.Cheque.CreatedAt,
		signedCheque.Cheque.ExpiresAt,
		signedCheque.Signature,
	)

	if err != nil {
		if err.Error() == "execution reverted" {
			return struct {
				Signer        common.Address
				PaymentAmount *big.Int
			}{}, nil
		}
		return struct {
			Signer        common.Address
			PaymentAmount *big.Int
		}{}, err
	}
	return result, nil
}

func (cm *evmChequeHandler) getLastCashIn(ctx context.Context, fromBot common.Address, toBot common.Address) (*LastCashIn, error) {
	callOpts := &bind.CallOpts{
		Context: ctx,
	}

	cashIn, err := cm.cmAccountInstance.GetLastCashIn(callOpts, fromBot, toBot)
	if err != nil {
		return nil, fmt.Errorf("failed to get last cash in: %w", err)
	}

	return &LastCashIn{
		Counter:   cashIn.LastCounter,
		Amount:    cashIn.LastAmount,
		CreatedAt: cashIn.LastCreatedAt,
		ExpiresAt: cashIn.LastExpiresAt,
	}, nil
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

func chequeRecordID(fromBot common.Address, toBot common.Address, toCmAccount common.Address) common.Hash {
	return crypto.Keccak256Hash(
		fromBot.Bytes(),
		toBot.Bytes(),
		toCmAccount.Bytes(),
	)
}
