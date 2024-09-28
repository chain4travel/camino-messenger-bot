package messaging

import (
	"context"
	"crypto/ecdsa"
	"errors"
	"fmt"
	"math/big"
	"time"

	"github.com/chain4travel/camino-messenger-bot/config"
	"github.com/chain4travel/camino-messenger-bot/internal/models"
	"github.com/chain4travel/camino-messenger-bot/internal/storage"
	"github.com/chain4travel/camino-messenger-bot/pkg/cheques"
	"github.com/chain4travel/camino-messenger-contracts/go/contracts/cmaccount"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethclient"
	lru "github.com/hashicorp/golang-lru/v2"
	"go.uber.org/zap"
)

var (
	_      ChequeHandler = (*evmChequeHandler)(nil)
	bigOne               = big.NewInt(1)
)

type evmChequeHandler struct {
	logger *zap.SugaredLogger

	chainID           *big.Int
	ethClient         *ethclient.Client
	evmConfig         *config.EvmConfig
	cmAccountAddress  common.Address
	cmAccountInstance *cmaccount.Cmaccount
	privateKey        *ecdsa.PrivateKey
	signer            cheques.Signer
	serviceRegistry   ServiceRegistry
	storage           storage.Storage
	cmAccounts        *lru.Cache[common.Address, *cmaccount.Cmaccount]
}

type ChequeHandler interface {
	IssueCheque(
		ctx context.Context,
		fromCmAccount common.Address,
		toCmAccount common.Address,
		fromBot common.Address,
		toBot common.Address,
		amount *big.Int,
	) (*cheques.SignedCheque, error)

	GetServiceFee(ctx context.Context, toCmAccountAddress common.Address, messageType MessageType) (*big.Int, error)
	IsBotAllowed(ctx context.Context, fromBot common.Address) (bool, error)
}

type LastCashIn struct {
	Counter   *big.Int
	Amount    *big.Int
	CreatedAt *big.Int
	ExpiresAt *big.Int
}

func NewChequeHandler(
	logger *zap.SugaredLogger,
	ethClient *ethclient.Client,
	evmConfig *config.EvmConfig,
	chainID *big.Int,
	storage storage.Storage,
	serviceRegistry ServiceRegistry,
) (ChequeHandler, error) {
	caminoPrivateKey, err := crypto.HexToECDSA(evmConfig.PrivateKey)
	if err != nil {
		return nil, err
	}

	cmAccountAddress := common.HexToAddress(evmConfig.CMAccountAddress)
	cmAccountInstance, err := cmaccount.NewCmaccount(cmAccountAddress, ethClient)
	if err != nil {
		return nil, fmt.Errorf("failed to instantiate contract binding: %w", err)
	}

	signer, err := cheques.NewSigner(caminoPrivateKey, chainID)
	if err != nil {
		return nil, fmt.Errorf("failed to create signer: %w", err)
	}

	cmAccountsCache, err := lru.New[common.Address, *cmaccount.Cmaccount](cmAccountsCacheSize)
	if err != nil {
		return nil, err
	}

	return &evmChequeHandler{
		ethClient:         ethClient,
		cmAccountAddress:  cmAccountAddress,
		cmAccountInstance: cmAccountInstance,
		chainID:           chainID,
		privateKey:        caminoPrivateKey,
		logger:            logger,
		evmConfig:         evmConfig,
		storage:           storage,
		signer:            signer,
		serviceRegistry:   serviceRegistry,
		cmAccounts:        cmAccountsCache,
	}, nil
}

func (ch *evmChequeHandler) IssueCheque(
	ctx context.Context,
	fromCMAccount common.Address,
	toCMAccount common.Address,
	fromBot common.Address,
	toBot common.Address,
	amount *big.Int,
) (*cheques.SignedCheque, error) {
	session, err := ch.storage.NewSession(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to create session: %w", err)
	}

	defer session.Abort()

	chequeRecordID := chequeRecordID(fromBot, toBot, toCMAccount)

	previousChequeModel, err := session.GetChequeRecord(ctx, chequeRecordID)
	if !errors.Is(err, storage.ErrNotFound) {
		return nil, fmt.Errorf("failed to get previous cheque: %w", err)
	}

	counter := big.NewInt(1)
	if previousChequeModel != nil {
		counter.Add(previousChequeModel.Counter, bigOne)
		amount.Add(previousChequeModel.Amount, amount)
	}

	now := time.Now().Unix()
	newCheque := &cheques.Cheque{
		FromCMAccount: fromCMAccount,
		ToCMAccount:   toCMAccount,
		ToBot:         toBot,
		Counter:       counter,
		Amount:        amount,
		CreatedAt:     big.NewInt(now),
		ExpiresAt:     big.NewInt(0).SetUint64(uint64(now) + ch.evmConfig.ChequeExpirationTime),
	}

	signedCheque, err := ch.signer.SignCheque(newCheque)
	if err != nil {
		return nil, fmt.Errorf("failed to sign cheque: %w", err)
	}

	if err := ch.verifyWithSmartContract(ctx, signedCheque); err != nil {
		return nil, fmt.Errorf("failed to verify cheque with smart contract: %w", err)
	}

	if err := session.UpsertChequeRecord(ctx, models.ChequeRecordFromCheque(chequeRecordID, signedCheque)); err != nil {
		ch.logger.Error(err)
		return nil, fmt.Errorf("failed to upsert cheque record: %w", err)
	}

	if err := session.Commit(); err != nil {
		ch.logger.Error(err)
		return nil, fmt.Errorf("failed to commit session: %w", err)
	}

	return signedCheque, nil
}

func (ch *evmChequeHandler) verifyWithSmartContract(ctx context.Context, signedCheque *cheques.SignedCheque) error {
	_, err := ch.cmAccountInstance.VerifyCheque(
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

	return err
}

func (ch *evmChequeHandler) GetServiceFee(ctx context.Context, toCmAccountAddress common.Address, messageType MessageType) (*big.Int, error) {
	supplierCmAccount, ok := ch.cmAccounts.Get(toCmAccountAddress)
	if !ok {
		var err error
		supplierCmAccount, err = cmaccount.NewCmaccount(toCmAccountAddress, ch.ethClient)
		if err != nil {
			ch.logger.Errorf("Failed to get cm Account: %v", err)
			return nil, err
		}
		ch.cmAccounts.Add(toCmAccountAddress, supplierCmAccount)
	}

	service, exists := servicesMapping[messageType]
	if !exists {
		return nil, fmt.Errorf("failed to get service identifier: %v", messageType)
	}

	serviceFee, err := supplierCmAccount.GetServiceFee(
		&bind.CallOpts{Context: ctx},
		service.servicePath,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to get service fee: %w", err)
	}
	return serviceFee, nil
}

func (ch *evmChequeHandler) IsBotAllowed(ctx context.Context, fromBot common.Address) (bool, error) {
	isAllowed, err := ch.cmAccountInstance.IsBotAllowed(&bind.CallOpts{Context: ctx}, fromBot)
	if err != nil {
		return false, fmt.Errorf("failed to check if bot has required permissions: %w", err)
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
