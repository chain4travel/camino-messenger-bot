package storage

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"math/big"

	"github.com/chain4travel/camino-messenger-bot/internal/models"
	"github.com/ethereum/go-ethereum/common"
	"github.com/jmoiron/sqlx"
)

const issuedChequeRecordsTableName = "issued_cheque_records"

var _ ChequeRecordsStorage = (*session)(nil)

type IssuedChequeRecordsStorage interface {
	GetIssuedChequeRecord(ctx context.Context, chequeRecordID common.Hash) (*models.IssuedChequeRecord, error)
	UpsertIssuedChequeRecord(ctx context.Context, chequeRecord *models.IssuedChequeRecord) error
}

type issuedChequeRecord struct {
	ChequeRecordID common.Hash `db:"cheque_record_id"`
	Counter        []byte      `db:"counter"`
	Amount         []byte      `db:"amount"`
}

func (s *session) GetIssuedChequeRecord(ctx context.Context, chequeRecordID common.Hash) (*models.IssuedChequeRecord, error) {
	chequeRecord := &issuedChequeRecord{}
	if err := s.tx.StmtxContext(ctx, s.storage.getIssuedChequeRecord).GetContext(ctx, chequeRecord, chequeRecordID); err != nil {
		if !errors.Is(err, sql.ErrNoRows) {
			s.logger.Error(err)
		}
		return nil, upgradeError(err)
	}
	return modelFromIssuedChequeRecord(chequeRecord), nil
}

func (s *session) UpsertIssuedChequeRecord(ctx context.Context, chequeRecord *models.IssuedChequeRecord) error {
	result, err := s.tx.NamedStmtContext(ctx, s.storage.upsertIssuedChequeRecord).
		ExecContext(ctx, issuedChequeRecordFromModel(chequeRecord))
	if err != nil {
		s.logger.Error(err)
		return upgradeError(err)
	}
	if rowsAffected, err := result.RowsAffected(); err != nil {
		s.logger.Error(err)
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
	getIssuedChequeRecord, err := s.db.PreparexContext(ctx, fmt.Sprintf(`
		SELECT * FROM %s
		WHERE cheque_record_id = ?
	`, issuedChequeRecordsTableName))
	if err != nil {
		s.logger.Error(err)
		return err
	}
	s.getIssuedChequeRecord = getIssuedChequeRecord

	upsertIssuedChequeRecord, err := s.db.PrepareNamedContext(ctx, fmt.Sprintf(`
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
		s.logger.Error(err)
		return err
	}
	s.upsertIssuedChequeRecord = upsertIssuedChequeRecord

	return nil
}

func modelFromIssuedChequeRecord(chequeRecord *issuedChequeRecord) *models.IssuedChequeRecord {
	return &models.IssuedChequeRecord{
		ChequeRecordID: chequeRecord.ChequeRecordID,
		Counter:        big.NewInt(0).SetBytes(chequeRecord.Counter),
		Amount:         big.NewInt(0).SetBytes(chequeRecord.Amount),
	}
}

func issuedChequeRecordFromModel(model *models.IssuedChequeRecord) *issuedChequeRecord {
	return &issuedChequeRecord{
		ChequeRecordID: model.ChequeRecordID,
		Counter:        model.Counter.Bytes(),
		Amount:         model.Amount.Bytes(),
	}
}
