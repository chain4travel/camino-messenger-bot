package sqlite

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"math/big"

	"github.com/chain4travel/camino-messenger-bot/pkg/chequehandler"
	"github.com/chain4travel/camino-messenger-bot/pkg/cheques"
	"github.com/ethereum/go-ethereum/common"
	"github.com/jmoiron/sqlx"
)

const chequeRecordsTableName = "cheque_records"

var (
	_ chequehandler.ChequeRecordsStorage = (*storage)(nil)

	zeroHash = common.Hash{}
)

type chequeRecord struct {
	ChequeRecordID common.Hash                   `db:"cheque_record_id"`
	FromCMAccount  common.Address                `db:"from_cm_account"`
	ToCMAccount    common.Address                `db:"to_cm_account"`
	ToBot          common.Address                `db:"to_bot"`
	Counter        []byte                        `db:"counter"`
	Amount         []byte                        `db:"amount"`
	CreatedAt      []byte                        `db:"created_at"`
	ExpiresAt      []byte                        `db:"expires_at"`
	Signature      []byte                        `db:"signature"`
	TxID           *common.Hash                  `db:"tx_id"`
	Status         *chequehandler.ChequeTxStatus `db:"status"`
}

func (s *storage) GetNotCashedChequeRecords(ctx context.Context, session chequehandler.Session) ([]*chequehandler.ChequeRecord, error) {
	tx, err := getSQLXTx(session)
	if err != nil {
		s.base.Logger.Error(err)
		return nil, err
	}

	chequeRecords := []*chequehandler.ChequeRecord{}
	rows, err := tx.StmtxContext(ctx, s.getNotCashedChequeRecords).QueryxContext(ctx)
	if err != nil {
		s.base.Logger.Error(err)
		return nil, upgradeError(err)
	}
	for rows.Next() {
		chequeRecord := &chequeRecord{}
		if err := rows.StructScan(chequeRecord); err != nil {
			s.base.Logger.Errorf("failed to get not cashed chequeRecord from db: %v", err)
			continue
		}
		model, err := modelFromChequeRecord(chequeRecord)
		if err != nil {
			s.base.Logger.Errorf("failed to parse not cashed chequeRecord: %v", err)
			continue
		}
		chequeRecords = append(chequeRecords, model)
	}
	return chequeRecords, nil
}

func (s *storage) GetChequeRecordsWithPendingTxs(ctx context.Context, session chequehandler.Session) ([]*chequehandler.ChequeRecord, error) {
	tx, err := getSQLXTx(session)
	if err != nil {
		s.base.Logger.Error(err)
		return nil, err
	}

	chequeRecords := []*chequehandler.ChequeRecord{}
	rows, err := tx.StmtxContext(ctx, s.getChequeRecordsWithPendingTxs).QueryxContext(ctx)
	if err != nil {
		s.base.Logger.Error(err)
		return nil, upgradeError(err)
	}
	for rows.Next() {
		chequeRecord := &chequeRecord{}
		if err := rows.StructScan(chequeRecord); err != nil {
			s.base.Logger.Errorf("failed to get chequeRecord with pending tx from db: %v", err)
			continue
		}
		model, err := modelFromChequeRecord(chequeRecord)
		if err != nil {
			s.base.Logger.Errorf("failed to parse chequeRecord with pending tx: %v", err)
			continue
		}
		chequeRecords = append(chequeRecords, model)
	}
	return chequeRecords, nil
}

func (s *storage) GetChequeRecord(ctx context.Context, session chequehandler.Session, chequeRecordID common.Hash) (*chequehandler.ChequeRecord, error) {
	tx, err := getSQLXTx(session)
	if err != nil {
		s.base.Logger.Error(err)
		return nil, err
	}

	chequeRecord := &chequeRecord{}
	if err := tx.StmtxContext(ctx, s.getChequeRecordByID).GetContext(ctx, chequeRecord, chequeRecordID); err != nil {
		if !errors.Is(err, sql.ErrNoRows) {
			s.base.Logger.Error(err)
		}
		return nil, upgradeError(err)
	}
	return modelFromChequeRecord(chequeRecord)
}

func (s *storage) GetChequeRecordByTxID(ctx context.Context, session chequehandler.Session, txID common.Hash) (*chequehandler.ChequeRecord, error) {
	tx, err := getSQLXTx(session)
	if err != nil {
		s.base.Logger.Error(err)
		return nil, err
	}

	chequeRecord := &chequeRecord{}
	if err := tx.StmtxContext(ctx, s.getChequeRecordByTxID).GetContext(ctx, chequeRecord, txID); err != nil {
		if !errors.Is(err, sql.ErrNoRows) {
			s.base.Logger.Error(err)
		}
		return nil, upgradeError(err)
	}
	return modelFromChequeRecord(chequeRecord)
}

func (s *storage) UpsertChequeRecord(ctx context.Context, session chequehandler.Session, chequeRecord *chequehandler.ChequeRecord) error {
	tx, err := getSQLXTx(session)
	if err != nil {
		s.base.Logger.Error(err)
		return err
	}

	result, err := tx.NamedStmtContext(ctx, s.upsertChequeRecord).
		ExecContext(ctx, chequeRecordFromModel(chequeRecord))
	if err != nil {
		s.base.Logger.Error(err)
		return upgradeError(err)
	}
	if rowsAffected, err := result.RowsAffected(); err != nil {
		s.base.Logger.Error(err)
		return upgradeError(err)
	} else if rowsAffected != 1 {
		return fmt.Errorf("failed to add chequeRecord: expected to affect 1 row, but affected %d", rowsAffected)
	}
	return nil
}

type chequeRecordsStatements struct {
	getNotCashedChequeRecords, getChequeRecordsWithPendingTxs *sqlx.Stmt
	getChequeRecordByID, getChequeRecordByTxID                *sqlx.Stmt
	upsertChequeRecord                                        *sqlx.NamedStmt
}

func (s *storage) prepareChequeRecordsStmts(ctx context.Context) error {
	getNotCashedChequeRecords, err := s.base.DB.PreparexContext(ctx, fmt.Sprintf(`
		SELECT * FROM %s
		WHERE status = %d OR status IS NULL
	`, chequeRecordsTableName, chequehandler.ChequeTxStatusRejected))
	if err != nil {
		s.base.Logger.Error(err)
		return err
	}
	s.getNotCashedChequeRecords = getNotCashedChequeRecords

	getChequeRecordsWithPendingTxs, err := s.base.DB.PreparexContext(ctx, fmt.Sprintf(`
		SELECT * FROM %s
		WHERE status = %d
	`, chequeRecordsTableName, chequehandler.ChequeTxStatusPending))
	if err != nil {
		s.base.Logger.Error(err)
		return err
	}
	s.getChequeRecordsWithPendingTxs = getChequeRecordsWithPendingTxs

	getChequeRecordByID, err := s.base.DB.PreparexContext(ctx, fmt.Sprintf(`
		SELECT * FROM %s
		WHERE cheque_record_id = ?
	`, chequeRecordsTableName))
	if err != nil {
		s.base.Logger.Error(err)
		return err
	}
	s.getChequeRecordByID = getChequeRecordByID

	getChequeByTxID, err := s.base.DB.PreparexContext(ctx, fmt.Sprintf(`
		SELECT * FROM %s
		WHERE tx_id = ?
	`, chequeRecordsTableName))
	if err != nil {
		s.base.Logger.Error(err)
		return err
	}
	s.getChequeRecordByTxID = getChequeByTxID

	upsertChequeRecord, err := s.base.DB.PrepareNamedContext(ctx, fmt.Sprintf(`
		INSERT INTO %s (
			cheque_record_id,
			from_cm_account,
			to_cm_account,
			to_bot,
			counter,
			amount,
			created_at,
			expires_at,
			signature,
			tx_id,
			status
		) VALUES (
			:cheque_record_id,
			:from_cm_account,
			:to_cm_account,
			:to_bot,
			:counter,
			:amount,
			:created_at,
			:expires_at,
			:signature,
			:tx_id,
			:status
		)
		ON CONFLICT(cheque_record_id)
		DO UPDATE SET 
			counter     = excluded.counter,
			amount      = excluded.amount,
			created_at  = excluded.created_at,
			expires_at  = excluded.expires_at,
			signature   = excluded.signature,
			tx_id       = excluded.tx_id,
			status      = excluded.status
	`, chequeRecordsTableName))
	if err != nil {
		s.base.Logger.Error(err)
		return err
	}
	s.upsertChequeRecord = upsertChequeRecord

	return nil
}

func modelFromChequeRecord(chequeRecord *chequeRecord) (*chequehandler.ChequeRecord, error) {
	txID := common.Hash{}
	if chequeRecord.TxID != nil {
		txID = *chequeRecord.TxID
	}

	status := chequehandler.ChequeTxStatusUnknown
	if chequeRecord.Status != nil {
		status = *chequeRecord.Status
	}

	return &chequehandler.ChequeRecord{
		SignedCheque: cheques.SignedCheque{
			Cheque: cheques.Cheque{
				FromCMAccount: chequeRecord.FromCMAccount,
				ToCMAccount:   chequeRecord.ToCMAccount,
				ToBot:         chequeRecord.ToBot,
				Counter:       big.NewInt(0).SetBytes(chequeRecord.Counter),
				Amount:        big.NewInt(0).SetBytes(chequeRecord.Amount),
				CreatedAt:     big.NewInt(0).SetBytes(chequeRecord.CreatedAt),
				ExpiresAt:     big.NewInt(0).SetBytes(chequeRecord.ExpiresAt),
			},
			Signature: chequeRecord.Signature,
		},
		ChequeRecordID: chequeRecord.ChequeRecordID,
		TxID:           txID,
		Status:         status,
	}, nil
}

func chequeRecordFromModel(model *chequehandler.ChequeRecord) *chequeRecord {
	var txID *common.Hash
	if model.TxID != zeroHash {
		txID = &model.TxID
	}

	var status *chequehandler.ChequeTxStatus
	if model.Status != chequehandler.ChequeTxStatusUnknown {
		status = &model.Status
	}

	return &chequeRecord{
		ChequeRecordID: model.ChequeRecordID,
		FromCMAccount:  model.FromCMAccount,
		ToCMAccount:    model.ToCMAccount,
		ToBot:          model.ToBot,
		Counter:        model.Counter.Bytes(),
		Amount:         model.Amount.Bytes(),
		CreatedAt:      model.CreatedAt.Bytes(),
		ExpiresAt:      model.ExpiresAt.Bytes(),
		Signature:      model.Signature,
		TxID:           txID,
		Status:         status,
	}
}
