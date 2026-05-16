// Copyright (C) 2022-2026, Chain4Travel AG. All rights reserved.
// See the file LICENSE for licensing terms.

package chequehandler

import (
	"context"
	"crypto/ecdsa"
	"errors"
	"fmt"
	"math/big"
	"sync"
	"time"

	"github.com/chain4travel/camino-messenger-bot/v13/pkg/cheques"
	cmaccounts "github.com/chain4travel/camino-messenger-bot/v13/pkg/cm_accounts"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"go.uber.org/zap"
	"golang.org/x/sync/errgroup"
)

var (
	_      ChequeHandler = (*evmChequeHandler)(nil)
	bigOne               = big.NewInt(1)

	ErrNotFound                 = errors.New("not found")
	errFailedToIssueValidCheque = errors.New("failed to issue valid cheque")
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
		toCmAccount common.Address,
		toBot common.Address,
		amount *big.Int,
	) (*cheques.SignedCheque, error)

	CashIn(ctx context.Context) error

	CheckCashInStatus(ctx context.Context) error

	VerifyAndStoreCheque(
		ctx context.Context,
		cheque *cheques.SignedCheque,
		sender common.Address,
		expectedAmountIncrement *big.Int,
	) error
}

func NewChequeHandler(
	ctx context.Context,
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
		return nil, fmt.Errorf("failed to create cheques signer: %w", err)
	}

	paymentToken, err := cmAccounts.GetServiceFeeToken(ctx, cmAccountAddress)
	if err != nil {
		return nil, fmt.Errorf("failed to get service fee token: %w", err)
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
		paymentToken:                     paymentToken,
	}, nil
}

type evmChequeHandler struct {
	logger *zap.SugaredLogger

	chainID                          *big.Int
	txReceiptGetter                  TxReceiptGetter
	cmAccountAddress                 common.Address    // cheque issuer, cheque recipient
	botKey                           *ecdsa.PrivateKey // cheque signer, cheque recipient
	botAddress                       common.Address
	signer                           cheques.Signer
	storage                          Storage
	cmAccounts                       cmaccounts.Service
	minChequeDurationUntilExpiration *big.Int
	chequeExpirationTime             *big.Int
	cashInTxIssueTimeout             time.Duration
	paymentToken                     common.Address
}

func (ch *evmChequeHandler) IssueCheque(
	ctx context.Context,
	toCMAccount common.Address,
	toBot common.Address,
	amount *big.Int,
) (*cheques.SignedCheque, error) {
	session, err := ch.storage.NewSession(ctx)
	if err != nil {
		err = fmt.Errorf("failed to create db session: %w", err)
		ch.logger.Error(err)
		return nil, err
	}
	defer ch.storage.Abort(session)

	now := big.NewInt(time.Now().Unix())
	newCheque := &cheques.Cheque{
		FromCMAccount: ch.cmAccountAddress,
		ToCMAccount:   toCMAccount,
		ToBot:         toBot,
		Counter:       big.NewInt(0),
		Amount:        big.NewInt(0).Set(amount),
		CreatedAt:     now,
		ExpiresAt:     big.NewInt(0).Add(now, ch.chequeExpirationTime),
		PaymentToken:  ch.paymentToken,
	}

	chequeRecordID := ChequeRecordID(newCheque)

	previousChequeModel, err := ch.storage.GetIssuedChequeRecord(ctx, session, chequeRecordID)
	if err != nil && !errors.Is(err, ErrNotFound) {
		err = fmt.Errorf("failed to get previous issued cheque record: %w", err)
		ch.logger.Error(err)
		return nil, err
	}

	if previousChequeModel != nil {
		newCheque.Counter.Add(previousChequeModel.Counter, bigOne)
		newCheque.Amount.Add(previousChequeModel.Amount, amount)
	}

	signedCheque, err := ch.signer.SignCheque(newCheque)
	if err != nil {
		err = fmt.Errorf("failed to sign new cheque: %w", err)
		ch.logger.Error(err)
		return nil, err
	}

	if isChequeValid, err := ch.cmAccounts.VerifyCheque(ctx, signedCheque); err != nil {
		err = fmt.Errorf("failed to verify cheque with smart contract: %w", err)
		ch.logger.Error(err)
		return nil, err
	} else if !isChequeValid {
		lastCounter, lastAmount, err := ch.cmAccounts.GetLastCashIn(ctx, ch.cmAccountAddress, ch.botAddress, toBot, newCheque.PaymentToken)
		if err != nil {
			err = fmt.Errorf("failed to get last cash in: %w", err)
			ch.logger.Error(err)
			return nil, err
		}
		newCheque.Counter.Add(lastCounter, bigOne)
		newCheque.Amount.Add(lastAmount, amount)

		signedCheque, err = ch.signer.SignCheque(newCheque)
		if err != nil {
			err = fmt.Errorf("failed to sign new cheque after getting last cash-in: %w", err)
			ch.logger.Error(err)
			return nil, err
		}

		if isChequeValid, err := ch.cmAccounts.VerifyCheque(ctx, signedCheque); err != nil {
			err = fmt.Errorf("failed to verify cheque with smart contract after getting last cash-in: %w", err)
			ch.logger.Error(err)
			return nil, err
		} else if !isChequeValid {
			ch.logger.Error(errFailedToIssueValidCheque)
			return nil, errFailedToIssueValidCheque
		}
	}

	if err := ch.storage.UpsertIssuedChequeRecord(ctx, session, IssuedChequeRecordCheque(chequeRecordID, signedCheque)); err != nil {
		err = fmt.Errorf("failed to upsert issued cheque record into db: %w", err)
		ch.logger.Error(err)
		return nil, err
	}

	if err := ch.storage.Commit(session); err != nil {
		err = fmt.Errorf("failed to commit db session: %w", err)
		ch.logger.Error(err)
		return nil, err
	}

	return signedCheque, nil
}

func (ch *evmChequeHandler) VerifyAndStoreCheque(
	ctx context.Context,
	cheque *cheques.SignedCheque,
	fromBot common.Address,
	expectedAmountIncrement *big.Int,
) error {
	if cheque.ToBot != ch.botAddress {
		return fmt.Errorf("cheque is not for this bot, expected %s, got %s", ch.botAddress.Hex(), cheque.ToBot.Hex())
	}
	if cheque.ToCMAccount != ch.cmAccountAddress {
		return fmt.Errorf("cheque is not for this CM account, expected %s, got %s", ch.cmAccountAddress.Hex(), cheque.ToCMAccount.Hex())
	}

	chequeIssuerPubKey, err := ch.signer.RecoverPublicKey(cheque)
	if err != nil {
		return fmt.Errorf("failed to recover cheque issuer public key: %w", err)
	}

	if fromBot != crypto.PubkeyToAddress(*chequeIssuerPubKey) {
		return fmt.Errorf("cheque issuer does not match sender")
	}

	session, err := ch.storage.NewSession(ctx)
	if err != nil {
		err = fmt.Errorf("failed to create db session: %w", err)
		ch.logger.Error(err)
		return err
	}
	defer ch.storage.Abort(session)

	chequeRecordID := ChequeRecordID(&cheque.Cheque)
	chequeRecord, err := ch.storage.GetChequeRecord(ctx, session, chequeRecordID)
	if err != nil && !errors.Is(err, ErrNotFound) {
		err = fmt.Errorf("failed to get chequeRecord from db: %w", err)
		ch.logger.Error(err)
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
		return fmt.Errorf("cheque verification failed: %w", err)
	}

	amountDiff := big.NewInt(0).Sub(cheque.Amount, oldAmount)
	if amountDiff.Cmp(expectedAmountIncrement) < 0 { // amountDiff < expectedAmountIncrement
		return fmt.Errorf("cheque amount must at least cover expectedAmountIncrement (%s), amountDiff is %s", expectedAmountIncrement.String(), amountDiff.String())
	}

	if valid, err := ch.cmAccounts.VerifyCheque(ctx, cheque); err != nil {
		err = fmt.Errorf("failed to verify cheque with blockchain: %w", err)
		ch.logger.Error(err)
		return err
	} else if !valid {
		return fmt.Errorf("cheque is invalid (blockchain validation)")
	}

	chequeRecord = ChequeRecordFromCheque(chequeRecordID, cheque)
	if err := ch.storage.UpsertChequeRecord(ctx, session, chequeRecord); err != nil {
		err = fmt.Errorf("failed to upsert chequeRecord into db: %w", err)
		ch.logger.Error(err)
		return err
	}

	if err := ch.storage.Commit(session); err != nil {
		err = fmt.Errorf("failed to commit db session: %w", err)
		ch.logger.Error(err)
		return err
	}

	return nil
}

func (ch *evmChequeHandler) CashIn(ctx context.Context) error {
	ch.logger.Debug("Cashing in...")
	defer ch.logger.Debug("Finished cashing in")

	session, err := ch.storage.NewSession(ctx)
	if err != nil {
		err = fmt.Errorf("failed to create db session: %w", err)
		ch.logger.Error(err)
		return err
	}
	defer ch.storage.Abort(session)

	chequeRecords, err := ch.storage.GetNotCashedChequeRecords(ctx, session)
	if err != nil {
		err = fmt.Errorf("failed to get not cashed cheques from db: %w", err)
		ch.logger.Error(err)
		return err
	}

	wg := sync.WaitGroup{}
	for _, chequeRecord := range chequeRecords {
		if chequeRecord.PaymentToken == (common.Address{}) {
			ch.logger.Debugf("Skipping cheque %s (no payment token)", chequeRecord)
			continue
		}

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
				ch.logger.Errorf("failed to cash in cheque %s: %v", chequeRecord, err)
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
		err = fmt.Errorf("failed to commit db session: %w", err)
		ch.logger.Error(err)
		return err
	}

	for _, chequeRecord := range chequeRecords {
		if chequeRecord.Status != ChequeTxStatusPending {
			continue
		}

		wg.Add(1)
		go func(txID common.Hash) {
			defer func() {
				if r := recover(); r != nil {
					ch.logger.Errorf("recovered from panic: checkCashInTxStatus for tx %s panicked: %v", txID.Hex(), r)
				}
				wg.Done()
			}()
			_ = ch.checkCashInTxStatus(ctx, txID)
		}(chequeRecord.TxID)
	}

	wg.Wait()

	return nil
}

func (ch *evmChequeHandler) CheckCashInStatus(ctx context.Context) error {
	session, err := ch.storage.NewSession(ctx)
	if err != nil {
		err = fmt.Errorf("failed to create db session: %w", err)
		ch.logger.Error(err)
		return err
	}
	defer ch.storage.Abort(session)

	chequeRecords, err := ch.storage.GetChequeRecordsWithPendingTxs(ctx, session)
	if err != nil {
		err = fmt.Errorf("failed to get not cashed cheques from db: %w", err)
		ch.logger.Error(err)
		return err
	}

	g := errgroup.Group{}

	for _, chequeRecord := range chequeRecords {
		txID := chequeRecord.TxID
		g.Go(func() (err error) {
			defer func() {
				if r := recover(); r != nil {
					err = fmt.Errorf("checkCashInTxStatus for tx %s panicked: %v", txID.Hex(), r) // err will be returned
					ch.logger.Errorf("recovered from panic: %v", err)
				}
			}()
			return ch.checkCashInTxStatus(ctx, txID)
		})
	}

	if err = g.Wait(); err != nil {
		err = fmt.Errorf("CheckCashInStatus failed with error: %w", err)
		ch.logger.Error(err)
	}

	return err
}

func (ch *evmChequeHandler) checkCashInTxStatus(ctx context.Context, txID common.Hash) error {
	// TODO @evlekht timeout? what to do if timeouted?
	res, err := ch.waitMined(ctx, txID)
	if err != nil {
		err = fmt.Errorf("failed to wait for tx %s to be mined: %w", txID.Hex(), err)
		ch.logger.Error(err)
		return err
	}

	session, err := ch.storage.NewSession(ctx)
	if err != nil {
		err = fmt.Errorf("failed to create db session: %w", err)
		ch.logger.Error(err)
		return err
	}
	defer ch.storage.Abort(session)

	chequeRecord, err := ch.storage.GetChequeRecordByTxID(ctx, session, txID)
	if err != nil {
		err = fmt.Errorf("failed to get cheque record by txID %s from db: %w", txID.Hex(), err)
		ch.logger.Error(err)
		return err
	}

	txStatus := ChequeTxStatusFromTxStatus(res.Status)
	if chequeRecord.Status == txStatus {
		return nil
	}

	chequeRecord.Status = txStatus
	if err := ch.storage.UpsertChequeRecord(ctx, session, chequeRecord); err != nil {
		err = fmt.Errorf("failed to update chequeRecord %s into db: %w", chequeRecord, err)
		ch.logger.Error(err)
		return err
	}

	if err := ch.storage.Commit(session); err != nil {
		err = fmt.Errorf("failed to commit db session: %w", err)
		ch.logger.Error(err)
		return err
	}

	return nil
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
