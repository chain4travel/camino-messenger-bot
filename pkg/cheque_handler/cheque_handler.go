package chequehandler

import (
	"context"
	"crypto/ecdsa"
	"errors"
	"fmt"
	"math/big"
	"sync"
	"time"

	"github.com/chain4travel/camino-messenger-bot/pkg/cheques"
	cmaccountscache "github.com/chain4travel/camino-messenger-bot/pkg/cm_accounts_cache"
	"github.com/chain4travel/camino-messenger-contracts/go/contracts/cmaccount"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	ethTypes "github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethclient"
	"go.uber.org/zap"
)

var (
	_      ChequeHandler = (*evmChequeHandler)(nil)
	bigOne               = big.NewInt(1)

	ErrNotFound = errors.New("not found")
)

type Storage interface {
	SessionHandler
	ChequeRecordsStorage
	IssuedChequeRecordsStorage
}

type ChequeRecordsStorage interface {
	GetNotCashedChequeRecords(ctx context.Context, session Session) ([]*ChequeRecord, error)
	GetChequeRecordsWithPendingTxs(ctx context.Context, session Session) ([]*ChequeRecord, error)
	GetChequeRecord(ctx context.Context, session Session, chequeRecordID common.Hash) (*ChequeRecord, error)
	GetChequeRecordByTxID(ctx context.Context, session Session, txID common.Hash) (*ChequeRecord, error)
	UpsertChequeRecord(ctx context.Context, session Session, chequeRecord *ChequeRecord) error
}

type IssuedChequeRecordsStorage interface {
	GetIssuedChequeRecord(ctx context.Context, session Session, chequeRecordID common.Hash) (*IssuedChequeRecord, error)
	UpsertIssuedChequeRecord(ctx context.Context, session Session, chequeRecord *IssuedChequeRecord) error
}

type SessionHandler interface {
	NewSession(ctx context.Context) (Session, error)
	Commit(session Session) error
	Abort(session Session)
}

type Session interface {
	Commit() error
	Abort() error
}

type ChequeHandler interface {
	IssueCheque(
		ctx context.Context,
		fromCmAccount common.Address,
		toCmAccount common.Address,
		toBot common.Address,
		amount *big.Int,
	) (*cheques.SignedCheque, error)

	IsAllowedToIssueCheque(ctx context.Context, fromBot common.Address) (bool, error)

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
	botKey *ecdsa.PrivateKey,
	cmAccountAddress common.Address,
	chainID *big.Int,
	storage Storage,
	cmAccounts cmaccountscache.CMAccountsCache,
	minChequeDurationUntilExpiration *big.Int,
	chequeExpirationTime *big.Int,
	cashInTxIssueTimeout time.Duration,
) (ChequeHandler, error) {
	cmAccountInstance, err := cmaccount.NewCmaccount(cmAccountAddress, ethClient)
	if err != nil {
		return nil, fmt.Errorf("failed to instantiate contract binding: %w", err)
	}

	signer, err := cheques.NewSigner(botKey, chainID)
	if err != nil {
		return nil, fmt.Errorf("failed to create signer: %w", err)
	}

	return &evmChequeHandler{
		ethClient:                        ethClient,
		cmAccountAddress:                 cmAccountAddress,
		cmAccountInstance:                cmAccountInstance,
		chainID:                          chainID,
		botKey:                           botKey,
		botAddress:                       crypto.PubkeyToAddress(botKey.PublicKey),
		logger:                           logger,
		storage:                          storage,
		signer:                           signer,
		cmAccounts:                       cmAccounts,
		minChequeDurationUntilExpiration: minChequeDurationUntilExpiration,
		chequeExpirationTime:             chequeExpirationTime,
		cashInTxIssueTimeout:             cashInTxIssueTimeout,
	}, nil
}

type evmChequeHandler struct {
	logger *zap.SugaredLogger

	chainID                          *big.Int
	ethClient                        *ethclient.Client
	cmAccountAddress                 common.Address
	cmAccountInstance                *cmaccount.Cmaccount
	botKey                           *ecdsa.PrivateKey
	botAddress                       common.Address
	signer                           cheques.Signer
	storage                          Storage
	cmAccounts                       cmaccountscache.CMAccountsCache
	minChequeDurationUntilExpiration *big.Int
	chequeExpirationTime             *big.Int
	cashInTxIssueTimeout             time.Duration
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

	defer ch.storage.Abort(session)

	now := big.NewInt(time.Now().Unix())
	newCheque := &cheques.Cheque{
		FromCMAccount: fromCMAccount,
		ToCMAccount:   toCMAccount,
		ToBot:         toBot,
		Counter:       big.NewInt(0),
		Amount:        big.NewInt(0).Set(amount),
		CreatedAt:     now,
		ExpiresAt:     big.NewInt(0).Add(now, ch.chequeExpirationTime),
	}

	chequeRecordID := ChequeRecordID(newCheque)

	previousChequeModel, err := ch.storage.GetIssuedChequeRecord(ctx, session, chequeRecordID)
	if err != nil && !errors.Is(err, ErrNotFound) {
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

	if err := ch.storage.UpsertIssuedChequeRecord(ctx, session, IssuedChequeRecordCheque(chequeRecordID, signedCheque)); err != nil {
		ch.logger.Error(err)
		return nil, fmt.Errorf("failed to upsert issued cheque record: %w", err)
	}

	if err := ch.storage.Commit(session); err != nil {
		ch.logger.Error(err)
		return nil, fmt.Errorf("failed to commit session: %w", err)
	}

	return signedCheque, nil
}

func (ch *evmChequeHandler) IsAllowedToIssueCheque(ctx context.Context, fromBot common.Address) (bool, error) {
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
	defer ch.storage.Abort(session)

	chequeIssuerPubKey, err := ch.signer.RecoverPublicKey(cheque)
	if err != nil {
		ch.logger.Errorf("failed to recover cheque issuer public key: %v", err)
		return err
	}

	if sender != crypto.PubkeyToAddress(*chequeIssuerPubKey) {
		return fmt.Errorf("cheque issuer does not match sender")
	}

	chequeRecordID := ChequeRecordID(&cheque.Cheque)
	chequeRecord, err := ch.storage.GetChequeRecord(ctx, session, chequeRecordID)
	if err != nil && !errors.Is(err, ErrNotFound) {
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
		ch.minChequeDurationUntilExpiration,
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

	chequeRecord = ChequeRecordFromCheque(chequeRecordID, cheque)
	if err := ch.storage.UpsertChequeRecord(ctx, session, chequeRecord); err != nil {
		ch.logger.Errorf("Failed to store cheque: %v", err)
		return err
	}

	return ch.storage.Commit(session)
}

func (ch *evmChequeHandler) verifyChequeWithContract(ctx context.Context, cheque *cheques.SignedCheque) (bool, error) {
	cmAccount, err := ch.cmAccounts.Get(cheque.FromCMAccount)
	if err != nil {
		ch.logger.Errorf("failed to get cmAccount contract instance: %v", err)
		return false, err
	}
	return verifyChequeWithContract(ctx, cmAccount, cheque)
}

func (ch *evmChequeHandler) CashIn(ctx context.Context) error {
	ch.logger.Debug("Cashing in...")
	defer ch.logger.Debug("Finished cashing in")

	session, err := ch.storage.NewSession(ctx)
	if err != nil {
		ch.logger.Error(err)
		return err
	}
	defer ch.storage.Abort(session)

	chequeRecords, err := ch.storage.GetNotCashedChequeRecords(ctx, session)
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

			timedCtx, cancel := context.WithTimeout(ctx, ch.cashInTxIssueTimeout)
			defer cancel()

			txID, err := ch.cashInCheque(timedCtx, chequeRecord)
			if err != nil {
				return
			}

			chequeRecord.TxID = txID
			chequeRecord.Status = ChequeTxStatusPending

			// TODO @evlekht if tx will be issued, but then storage will fail to persist it,
			// TODO tx is still issued and app service will fail to cash in this cheque next time
			// TODO cause on the node side it is already cashed in
			// TODO possible solution would be to do dry run, get txID, commit session with txID and status "processing",
			// TODO then do real run? also do same on startup

			// TODO @evlekht add txCreatedAt field to db and use it for mining timeout ?

			if err := ch.storage.UpsertChequeRecord(ctx, session, chequeRecord); err != nil {
				chequeRecord.Status = ChequeTxStatusUnknown
				ch.logger.Errorf("failed to update cheque %s: %v", chequeRecord, err)
				return
			}
		}()
	}

	wg.Wait()

	if err := ch.storage.Commit(session); err != nil {
		ch.logger.Errorf("failed to commit session: %v", err)
		return err
	}

	for _, chequeRecord := range chequeRecords {
		if chequeRecord.Status != ChequeTxStatusPending {
			continue
		}

		go func(txID common.Hash) {
			_ = ch.checkCashInStatus(context.Background(), txID)
		}(chequeRecord.TxID)
	}

	return nil
}

func (ch *evmChequeHandler) cashInCheque(ctx context.Context, chequeRecord *ChequeRecord) (common.Hash, error) {
	cmAccount, err := ch.cmAccounts.Get(chequeRecord.FromCMAccount)
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
	defer ch.storage.Abort(session)

	chequeRecords, err := ch.storage.GetChequeRecordsWithPendingTxs(ctx, session)
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
	defer ch.storage.Abort(session)

	chequeRecord, err := ch.storage.GetChequeRecordByTxID(ctx, session, txID)
	if err != nil {
		ch.logger.Errorf("failed to get chequeRecord by txID %s: %v", txID, err)
		return err
	}

	txStatus := ChequeTxStatusFromTxStatus(res.Status)
	if chequeRecord.Status == txStatus {
		return nil
	}

	chequeRecord.Status = txStatus
	if err := ch.storage.UpsertChequeRecord(ctx, session, chequeRecord); err != nil {
		ch.logger.Errorf("failed to update chequeRecord %s: %v", chequeRecord, err)
		return err
	}

	return ch.storage.Commit(session)
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

func waitMined(ctx context.Context, b bind.DeployBackend, txID common.Hash) (*ethTypes.Receipt, error) {
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
