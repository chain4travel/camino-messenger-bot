package sqlite

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"math/big"

	chequeHandler "github.com/chain4travel/camino-messenger-bot/pkg/cheque_handler"
	"github.com/ethereum/go-ethereum/common"
	"github.com/jmoiron/sqlx"
)

const issuedChequeRecordsTableName = "issued_cheque_records"

var _ chequeHandler.ChequeRecordsStorage = (*storage)(nil)

type issuedChequeRecord struct {
	ChequeRecordID common.Hash `db:"cheque_record_id"`
	Counter        []byte      `db:"counter"`
	Amount         []byte      `db:"amount"`
}

func (s *storage) GetIssuedChequeRecord(ctx context.Context, session chequeHandler.Session, chequeRecordID common.Hash) (*chequeHandler.IssuedChequeRecord, error) {
	tx, err := getSQLXTx(session)
	if err != nil {
		s.base.Logger.Error(err)
		return nil, err
	}

	chequeRecord := &issuedChequeRecord{}
	if err := tx.StmtxContext(ctx, s.getIssuedChequeRecord).GetContext(ctx, chequeRecord, chequeRecordID); err != nil {
		if !errors.Is(err, sql.ErrNoRows) {
			s.base.Logger.Error(err)
		}
		return nil, upgradeError(err)
	}
	return modelFromIssuedChequeRecord(chequeRecord), nil
}

func (s *storage) UpsertIssuedChequeRecord(ctx context.Context, session chequeHandler.Session, chequeRecord *chequeHandler.IssuedChequeRecord) error {
	tx, err := getSQLXTx(session)
	if err != nil {
		s.base.Logger.Error(err)
		return err
	}

	result, err := tx.NamedStmtContext(ctx, s.upsertIssuedChequeRecord).
		ExecContext(ctx, issuedChequeRecordFromModel(chequeRecord))
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

type issuedChequeRecordsStatements struct {
	getIssuedChequeRecord    *sqlx.Stmt
	upsertIssuedChequeRecord *sqlx.NamedStmt
}

func (s *storage) prepareIssuedChequeRecordsStmts(ctx context.Context) error {
	getIssuedChequeRecord, err := s.base.DB.PreparexContext(ctx, fmt.Sprintf(`
		SELECT * FROM %s
		WHERE cheque_record_id = ?
	`, issuedChequeRecordsTableName))
	if err != nil {
		s.base.Logger.Error(err)
		return err
	}
	s.getIssuedChequeRecord = getIssuedChequeRecord

	upsertIssuedChequeRecord, err := s.base.DB.PrepareNamedContext(ctx, fmt.Sprintf(`
		INSERT INTO %s (
			cheque_record_id,
			counter,
			amount
		) VALUES (
			:cheque_record_id,
			:counter,
			:amount
		)
		ON CONFLICT(cheque_record_id)
		DO UPDATE SET
			counter = excluded.counter,
			amount  = excluded.amount
	`, issuedChequeRecordsTableName))
	if err != nil {
		s.base.Logger.Error(err)
		return err
	}
	s.upsertIssuedChequeRecord = upsertIssuedChequeRecord

	return nil
}

func modelFromIssuedChequeRecord(chequeRecord *issuedChequeRecord) *chequeHandler.IssuedChequeRecord {
	return &chequeHandler.IssuedChequeRecord{
		ChequeRecordID: chequeRecord.ChequeRecordID,
		Counter:        big.NewInt(0).SetBytes(chequeRecord.Counter),
		Amount:         big.NewInt(0).SetBytes(chequeRecord.Amount),
	}
}

func issuedChequeRecordFromModel(model *chequeHandler.IssuedChequeRecord) *issuedChequeRecord {
	return &issuedChequeRecord{
		ChequeRecordID: model.ChequeRecordID,
		Counter:        model.Counter.Bytes(),
		Amount:         model.Amount.Bytes(),
	}
}
