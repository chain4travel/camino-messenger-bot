package messaging

import (
	"context"
	"crypto/ecdsa"
	"errors"
	"fmt"
	"math/big"
	"sync"
	"time"

	"github.com/chain4travel/camino-messenger-bot/config"
	"github.com/chain4travel/camino-messenger-bot/internal/messaging/messages"
	"github.com/chain4travel/camino-messenger-bot/internal/models"
	"github.com/chain4travel/camino-messenger-bot/internal/storage"
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
	_      ChequeHandler = (*evmChequeHandler)(nil)
	bigOne               = big.NewInt(1)
)

type ChequeHandler interface {
	IssueCheque(
		ctx context.Context,
		fromCmAccount common.Address,
		toCmAccount common.Address,
		toBot common.Address,
		amount *big.Int,
	) (*cheques.SignedCheque, error)

	GetServiceFee(
		ctx context.Context,
		toCmAccountAddress common.Address,
		messageType messages.MessageType,
	) (*big.Int, error)

	IsBotAllowed(ctx context.Context, fromBot common.Address) (bool, error)

	CashIn(ctx context.Context) error

	CheckCashInStatus(ctx context.Context) error

	VerifyCheque(
		ctx context.Context,
		cheque *cheques.SignedCheque,
		sender common.Address,
		serviceFee *big.Int,
	) error
}

func NewChequeHandler(
	logger *zap.SugaredLogger,
	ethClient *ethclient.Client,
	evmConfig *config.EvmConfig,
	chainID *big.Int,
	storage storage.Storage,
	serviceRegistry ServiceRegistry,
) (ChequeHandler, error) {
	botKey, err := crypto.HexToECDSA(evmConfig.PrivateKey)
	if err != nil {
		return nil, err
	}

	cmAccountAddress := common.HexToAddress(evmConfig.CMAccountAddress)
	cmAccountInstance, err := cmaccount.NewCmaccount(cmAccountAddress, ethClient)
	if err != nil {
		return nil, fmt.Errorf("failed to instantiate contract binding: %w", err)
	}

	signer, err := cheques.NewSigner(botKey, chainID)
	if err != nil {
		return nil, fmt.Errorf("failed to create signer: %w", err)
	}

	cmAccountsCache, err := lru.New[common.Address, *cmaccount.Cmaccount](cmAccountsCacheSize)
	if err != nil {
		return nil, err
	}

	return &evmChequeHandler{
		ethClient:                  ethClient,
		cmAccountAddress:           cmAccountAddress,
		cmAccountInstance:          cmAccountInstance,
		chainID:                    chainID,
		botKey:                     botKey,
		botAddress:                 crypto.PubkeyToAddress(botKey.PublicKey),
		logger:                     logger,
		evmConfig:                  evmConfig,
		storage:                    storage,
		signer:                     signer,
		serviceRegistry:            serviceRegistry,
		cmAccounts:                 cmAccountsCache,
		minDurationUntilExpiration: big.NewInt(0).SetUint64(evmConfig.MinChequeDurationUntilExpiration),
	}, nil
}

type evmChequeHandler struct {
	logger *zap.SugaredLogger

	chainID                    *big.Int
	ethClient                  *ethclient.Client
	evmConfig                  *config.EvmConfig
	cmAccountAddress           common.Address
	cmAccountInstance          *cmaccount.Cmaccount
	botKey                     *ecdsa.PrivateKey
	botAddress                 common.Address
	signer                     cheques.Signer
	serviceRegistry            ServiceRegistry
	storage                    storage.Storage
	cmAccounts                 *lru.Cache[common.Address, *cmaccount.Cmaccount]
	minDurationUntilExpiration *big.Int
}

func (ch *evmChequeHandler) IssueCheque(
	ctx context.Context,
	fromCMAccount common.Address,
	toCMAccount common.Address,
	toBot common.Address,
	amount *big.Int,
) (*cheques.SignedCheque, error) {
	session, err := ch.storage.NewSession(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to create session: %w", err)
	}

	defer session.Abort()

	now := time.Now().Unix()
	newCheque := &cheques.Cheque{
		FromCMAccount: fromCMAccount,
		ToCMAccount:   toCMAccount,
		ToBot:         toBot,
		Counter:       big.NewInt(0),
		Amount:        big.NewInt(0).Set(amount),
		CreatedAt:     big.NewInt(now),
		ExpiresAt:     big.NewInt(0).SetUint64(uint64(now) + ch.evmConfig.ChequeExpirationTime),
	}

	chequeRecordID := models.ChequeRecordID(newCheque)

	previousChequeModel, err := session.GetIssuedChequeRecord(ctx, chequeRecordID)
	if err != nil && !errors.Is(err, storage.ErrNotFound) {
		ch.logger.Errorf("failed to get previous cheque: %v", err)
		return nil, fmt.Errorf("failed to get previous cheque: %w", err)
	}

	if previousChequeModel != nil {
		newCheque.Counter.Add(previousChequeModel.Counter, bigOne)
		newCheque.Amount.Add(previousChequeModel.Amount, amount)
	}

	signedCheque, err := ch.signer.SignCheque(newCheque)
	if err != nil {
		ch.logger.Errorf("failed to sign cheque: %v", err)
		return nil, fmt.Errorf("failed to sign cheque: %w", err)
	}

	isChequeValid, err := verifyChequeWithContract(ctx, ch.cmAccountInstance, signedCheque)
	if err != nil {
		ch.logger.Errorf("failed to verify cheque with smart contract: %v", err)
		return nil, fmt.Errorf("failed to verify cheque with smart contract: %w", err)
	} else if !isChequeValid {
		lastCounter, lastAmount, err := ch.getLastCashIn(ctx, toBot)
		if err != nil {
			ch.logger.Errorf("failed to get last cash in: %v", err)
			return nil, fmt.Errorf("failed to get last cash in: %w", err)
		}
		newCheque.Counter.Add(lastCounter, bigOne)
		newCheque.Amount.Add(lastAmount, amount)

		signedCheque, err = ch.signer.SignCheque(newCheque)
		if err != nil {
			ch.logger.Errorf("failed to sign cheque: %v", err)
			return nil, fmt.Errorf("failed to sign cheque: %w", err)
		}

		isChequeValid, err := verifyChequeWithContract(ctx, ch.cmAccountInstance, signedCheque)
		if err != nil {
			ch.logger.Errorf("failed to verify cheque with smart contract after getting last cash-in: %v", err)
			return nil, fmt.Errorf("failed to verify cheque with smart contract: %w", err)
		} else if !isChequeValid {
			ch.logger.Errorf("failed to issue valid cheque")
			return nil, fmt.Errorf("failed to issue valid cheque")
		}
	}

	if err := session.UpsertIssuedChequeRecord(ctx, models.IssuedChequeRecordCheque(chequeRecordID, signedCheque)); err != nil {
		ch.logger.Error(err)
		return nil, fmt.Errorf("failed to upsert issued cheque record: %w", err)
	}

	if err := session.Commit(); err != nil {
		ch.logger.Error(err)
		return nil, fmt.Errorf("failed to commit session: %w", err)
	}

	return signedCheque, nil
}

func (ch *evmChequeHandler) GetServiceFee(ctx context.Context, toCmAccountAddress common.Address, messageType messages.MessageType) (*big.Int, error) {
	supplierCmAccount, err := ch.getCMAccount(toCmAccountAddress)
	if err != nil {
		return nil, fmt.Errorf("failed to get supplier cmAccount: %w", err)
	}

	serviceFullName, err := getServiceFullName(messageType)
	if err != nil {
		return nil, fmt.Errorf("failed to get service: %v", err)
	}

	serviceFee, err := supplierCmAccount.GetServiceFee(
		&bind.CallOpts{Context: ctx},
		serviceFullName,
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

func (ch *evmChequeHandler) VerifyCheque(
	ctx context.Context,
	cheque *cheques.SignedCheque,
	sender common.Address,
	serviceFee *big.Int,
) error {
	session, err := ch.storage.NewSession(ctx)
	if err != nil {
		ch.logger.Errorf("failed to create storage session: %v", err)
		return err
	}
	defer session.Abort()

	chequeIssuerPubKey, err := ch.signer.RecoverPublicKey(cheque)
	if err != nil {
		ch.logger.Errorf("failed to recover cheque issuer public key: %v", err)
		return err
	}

	if sender != crypto.PubkeyToAddress(*chequeIssuerPubKey) {
		return fmt.Errorf("cheque issuer does not match sender")
	}

	chequeRecordID := models.ChequeRecordID(&cheque.Cheque)
	chequeRecord, err := session.GetChequeRecord(ctx, chequeRecordID)
	if err != nil && !errors.Is(err, storage.ErrNotFound) {
		ch.logger.Errorf("failed to get chequeRecord: %v", err)
		return err
	}

	var previousCheque *cheques.SignedCheque
	oldAmount := big.NewInt(0)
	if chequeRecord != nil {
		previousCheque = &chequeRecord.SignedCheque
		oldAmount.Set(chequeRecord.Amount)
	}
	if err := cheques.VerifyCheque(
		previousCheque,
		cheque,
		big.NewInt(time.Now().Unix()),
		ch.minDurationUntilExpiration,
	); err != nil {
		return err
	}

	amountDiff := big.NewInt(0).Sub(cheque.Amount, oldAmount)
	if amountDiff.Cmp(serviceFee) < 0 { // amountDiff < serviceFee
		return fmt.Errorf("cheque amount must at least cover serviceFee")
	}

	if valid, err := ch.verifyChequeWithContract(ctx, cheque); err != nil {
		ch.logger.Errorf("Failed to verify cheque with blockchain: %v", err)
		return err
	} else if !valid {
		ch.logger.Infof("cheque is invalid (blockchain validation)")
		return fmt.Errorf("cheque is invalid (blockchain validation)")
	}

	chequeRecord = models.ChequeRecordFromCheque(chequeRecordID, cheque)
	if err := session.UpsertChequeRecord(ctx, chequeRecord); err != nil {
		ch.logger.Errorf("Failed to store cheque: %v", err)
		return err
	}

	return session.Commit()
}

func (ch *evmChequeHandler) verifyChequeWithContract(ctx context.Context, cheque *cheques.SignedCheque) (bool, error) {
	cmAccount, err := ch.getCMAccount(cheque.FromCMAccount)
	if err != nil {
		ch.logger.Errorf("failed to get cmAccount contract instance: %v", err)
		return false, err
	}
	return verifyChequeWithContract(ctx, cmAccount, cheque)
}

// TODO @evlekht whole cash in is almost 100% copy-paste from asb, think of moving to common place
func (ch *evmChequeHandler) CashIn(ctx context.Context) error {
	ch.logger.Debug("Cashing in...")
	defer ch.logger.Debug("Finished cashing in")

	session, err := ch.storage.NewSession(ctx)
	if err != nil {
		ch.logger.Error(err)
		return err
	}
	defer session.Abort()

	chequeRecords, err := session.GetNotCashedChequeRecords(ctx)
	if err != nil {
		ch.logger.Errorf("failed to get not cashed cheques: %v", err)
		return err
	}

	wg := sync.WaitGroup{}
	for _, chequeRecord := range chequeRecords {
		ch.logger.Debugf("Checking cheque %s status...", chequeRecord)

		wg.Add(1)
		go func() {
			defer wg.Done()

			timedCtx, cancel := context.WithTimeout(ctx, cashInTxIssueTimeout)
			defer cancel()

			txID, err := ch.cashInCheque(timedCtx, chequeRecord)
			if err != nil {
				return
			}

			chequeRecord.TxID = txID
			chequeRecord.Status = models.ChequeTxStatusPending

			// TODO @evlekht if tx will be issued, but then storage will fail to persist it,
			// TODO tx is still issued and app service will fail to cash in this cheque next time
			// TODO cause on the node side it is already cashed in
			// TODO possible solution would be to do dry run, get txID, commit session with txID and status "processing",
			// TODO then do real run? also do same on startup

			// TODO @evlekht add txCreatedAt field to db and use it for mining timeout ?

			if err := session.UpsertChequeRecord(ctx, chequeRecord); err != nil {
				chequeRecord.Status = models.ChequeTxStatusUnknown
				ch.logger.Errorf("failed to update cheque %s: %v", chequeRecord, err)
				return
			}
		}()
	}

	wg.Wait()

	if err := session.Commit(); err != nil {
		ch.logger.Errorf("failed to commit session: %v", err)
		return err
	}

	for _, chequeRecord := range chequeRecords {
		if chequeRecord.Status != models.ChequeTxStatusPending {
			continue
		}

		go func(txID common.Hash) {
			_ = ch.checkCashInStatus(context.Background(), txID)
		}(chequeRecord.TxID)
	}

	return nil
}

func (ch *evmChequeHandler) cashInCheque(ctx context.Context, chequeRecord *models.ChequeRecord) (common.Hash, error) {
	cmAccount, err := ch.getCMAccount(chequeRecord.FromCMAccount)
	if err != nil {
		ch.logger.Errorf("failed to get cmAccount contract instance: %v", err)
		return common.Hash{}, err
	}

	transactor, err := bind.NewKeyedTransactorWithChainID(ch.botKey, ch.chainID)
	if err != nil {
		ch.logger.Error(err)
		return common.Hash{}, err
	}
	transactor.Context = ctx

	tx, err := cmAccount.CashInCheque(
		transactor,
		chequeRecord.FromCMAccount,
		chequeRecord.ToCMAccount,
		chequeRecord.ToBot,
		chequeRecord.Counter,
		chequeRecord.Amount,
		chequeRecord.CreatedAt,
		chequeRecord.ExpiresAt,
		chequeRecord.Signature,
	)
	if err != nil {
		ch.logger.Errorf("failed to cash in cheque %s: %v", chequeRecord, err)
		return common.Hash{}, err
	}

	return tx.Hash(), nil
}

func (ch *evmChequeHandler) CheckCashInStatus(ctx context.Context) error {
	session, err := ch.storage.NewSession(ctx)
	if err != nil {
		ch.logger.Error(err)
		return err
	}
	defer session.Abort()

	chequeRecords, err := session.GetChequeRecordsWithPendingTxs(ctx)
	if err != nil {
		ch.logger.Errorf("failed to get not cashed cheques: %v", err)
		return err
	}

	for _, chequeRecord := range chequeRecords {
		go func(txID common.Hash) {
			_ = ch.checkCashInStatus(ctx, txID)
		}(chequeRecord.TxID)
	}

	return nil
}

func (ch *evmChequeHandler) checkCashInStatus(ctx context.Context, txID common.Hash) error {
	// TODO @evlekht timeout? what to do if timeouted?
	res, err := waitMined(ctx, ch.ethClient, txID)
	if err != nil {
		ch.logger.Errorf("failed to get cash in transaction receipt %s: %v", txID, err)
		return err
	}

	session, err := ch.storage.NewSession(ctx)
	if err != nil {
		ch.logger.Error(err)
		return err
	}
	defer session.Abort()

	chequeRecord, err := session.GetChequeRecordByTxID(ctx, txID)
	if err != nil {
		ch.logger.Errorf("failed to get chequeRecord by txID %s: %v", txID, err)
		return err
	}

	txStatus := models.ChequeTxStatusFromTxStatus(res.Status)
	if chequeRecord.Status == txStatus {
		return nil
	}

	chequeRecord.Status = txStatus
	if err := session.UpsertChequeRecord(ctx, chequeRecord); err != nil {
		ch.logger.Errorf("failed to update chequeRecord %s: %v", chequeRecord, err)
		return err
	}

	return session.Commit()
}

func (ch *evmChequeHandler) getCMAccount(address common.Address) (*cmaccount.Cmaccount, error) {
	cmAccount, ok := ch.cmAccounts.Get(address)
	if ok {
		return cmAccount, nil
	}

	cmAccount, err := cmaccount.NewCmaccount(address, ch.ethClient)
	if err != nil {
		ch.logger.Errorf("failed to create cmAccount contract instance: %v", err)
		return nil, err
	}
	_ = ch.cmAccounts.Add(address, cmAccount)

	return cmAccount, nil
}

func (ch *evmChequeHandler) getLastCashIn(ctx context.Context, toBot common.Address) (counter *big.Int, amount *big.Int, err error) {
	lastCashIn, err := ch.cmAccountInstance.GetLastCashIn(
		&bind.CallOpts{Context: ctx},
		ch.botAddress,
		toBot,
	)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to get last cash in: %w", err)
	}
	return lastCashIn.LastCounter, lastCashIn.LastAmount, nil
}

func verifyChequeWithContract(
	ctx context.Context,
	cmAcc *cmaccount.Cmaccount,
	signedCheque *cheques.SignedCheque,
) (bool, error) {
	_, err := cmAcc.VerifyCheque(
		&bind.CallOpts{Context: ctx},
		signedCheque.FromCMAccount,
		signedCheque.ToCMAccount,
		signedCheque.ToBot,
		signedCheque.Counter,
		signedCheque.Amount,
		signedCheque.CreatedAt,
		signedCheque.ExpiresAt,
		signedCheque.Signature,
	)
	if err != nil && err.Error() == "execution reverted" {
		return false, nil
	}
	return err == nil, err
}

func waitMined(ctx context.Context, b bind.DeployBackend, txID common.Hash) (*types.Receipt, error) {
	ticker := time.NewTicker(time.Second)
	defer ticker.Stop()

	for {
		receipt, err := b.TransactionReceipt(ctx, txID)
		if err == nil {
			return receipt, nil
		}

		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-ticker.C:
		}
	}
}
