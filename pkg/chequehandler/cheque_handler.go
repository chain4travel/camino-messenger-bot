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
	cmaccounts "github.com/chain4travel/camino-messenger-bot/pkg/cm_accounts"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
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

type TxReceiptGetter interface {
	TransactionReceipt(ctx context.Context, txHash common.Hash) (*types.Receipt, error)
}

type ChequeHandler interface {
	IssueCheque(
		ctx context.Context,
		fromCmAccount common.Address,
		toCmAccount common.Address,
		toBot common.Address,
		amount *big.Int,
	) (*cheques.SignedCheque, error)

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
	ethClient TxReceiptGetter,
	botKey *ecdsa.PrivateKey,
	cmAccountAddress common.Address,
	chainID *big.Int,
	storage Storage,
	cmAccounts cmaccounts.Service,
	minChequeDurationUntilExpiration *big.Int,
	chequeExpirationTime *big.Int,
	cashInTxIssueTimeout time.Duration,
) (ChequeHandler, error) {
	signer, err := cheques.NewSigner(botKey, chainID)
	if err != nil {
		return nil, fmt.Errorf("failed to create signer: %w", err)
	}

	return &evmChequeHandler{
		txReceiptGetter:                  ethClient,
		cmAccountAddress:                 cmAccountAddress,
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
	txReceiptGetter                  TxReceiptGetter
	cmAccountAddress                 common.Address
	botKey                           *ecdsa.PrivateKey
	botAddress                       common.Address
	signer                           cheques.Signer
	storage                          Storage
	cmAccounts                       cmaccounts.Service
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

	if isChequeValid, err := ch.cmAccounts.VerifyCheque(ctx, signedCheque); err != nil {
		ch.logger.Errorf("failed to verify cheque with smart contract: %v", err)
		return nil, fmt.Errorf("failed to verify cheque with smart contract: %w", err)
	} else if !isChequeValid {
		lastCounter, lastAmount, err := ch.cmAccounts.GetLastCashIn(ctx, ch.cmAccountAddress, ch.botAddress, toBot)
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

		if isChequeValid, err := ch.cmAccounts.VerifyCheque(ctx, signedCheque); err != nil {
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

	if valid, err := ch.cmAccounts.VerifyCheque(ctx, cheque); err != nil {
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

			txID, err := ch.cmAccounts.CashInCheque(
				timedCtx,
				&chequeRecord.SignedCheque,
				ch.botKey,
			)
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
	res, err := ch.waitMined(ctx, txID)
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

func (ch *evmChequeHandler) waitMined(ctx context.Context, txID common.Hash) (*types.Receipt, error) {
	ticker := time.NewTicker(time.Second)
	defer ticker.Stop()

	for {
		receipt, err := ch.txReceiptGetter.TransactionReceipt(ctx, txID)
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
