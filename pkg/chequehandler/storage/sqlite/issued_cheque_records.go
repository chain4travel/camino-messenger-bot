// Copyright (C) 2022-2026, Chain4Travel AG. All rights reserved.
// See the file LICENSE for licensing terms.

package sqlite

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"math/big"

	"github.com/chain4travel/camino-messenger-bot/v13/pkg/chequehandler"
	"github.com/chain4travel/camino-messenger-bot/v13/pkg/database/sqlite"
	"github.com/ethereum/go-ethereum/common"
	"github.com/jmoiron/sqlx"
)

const issuedChequeRecordsTableName = "issued_cheque_records"

var _ chequehandler.ChequeRecordsStorage = (*storage)(nil)

type issuedChequeRecord struct {
	ChequeRecordID common.Hash `db:"cheque_record_id"`
	Counter        []byte      `db:"counter"`
	Amount         []byte      `db:"amount"`
}

func (s *storage) GetIssuedChequeRecord(ctx context.Context, session chequehandler.Session, chequeRecordID common.Hash) (*chequehandler.IssuedChequeRecord, error) {
	tx, err := sqlite.GetSQLXTx(session)
	if err != nil {
		return nil, fmt.Errorf("failed to get transaction from db session: %w", err)
	}

	chequeRecord := &issuedChequeRecord{}
	if err := tx.StmtxContext(ctx, s.getIssuedChequeRecord).GetContext(ctx, chequeRecord, chequeRecordID); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, chequehandler.ErrNotFound
		}
		return nil, fmt.Errorf("failed to execute get issued cheque record by ID statement: %w", err)
	}
	return modelFromIssuedChequeRecord(chequeRecord), nil
}

func (s *storage) UpsertIssuedChequeRecord(ctx context.Context, session chequehandler.Session, chequeRecord *chequehandler.IssuedChequeRecord) error {
	tx, err := sqlite.GetSQLXTx(session)
	if err != nil {
		return fmt.Errorf("failed to get transaction from db session: %w", err)
	}

	result, err := tx.NamedStmtContext(ctx, s.upsertIssuedChequeRecord).
		ExecContext(ctx, issuedChequeRecordFromModel(chequeRecord))
	if err != nil {
		return fmt.Errorf("failed to execute upsert issued cheque record statement: %w", err)
	}
	if rowsAffected, err := result.RowsAffected(); err != nil {
		return fmt.Errorf("failed to get rowsAffected from statement execution result: %w", err)
	} else if rowsAffected != 1 {
		return fmt.Errorf("unexpected number of rows affected: expected 1, but affected %d", rowsAffected)
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
		return fmt.Errorf("failed to prepare get issued cheque record statement: %w", err)
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
		return fmt.Errorf("failed to prepare upsert issued cheque record statement: %w", err)
	}
	s.upsertIssuedChequeRecord = upsertIssuedChequeRecord

	return nil
}

func modelFromIssuedChequeRecord(chequeRecord *issuedChequeRecord) *chequehandler.IssuedChequeRecord {
	return &chequehandler.IssuedChequeRecord{
		ChequeRecordID: chequeRecord.ChequeRecordID,
		Counter:        big.NewInt(0).SetBytes(chequeRecord.Counter),
		Amount:         big.NewInt(0).SetBytes(chequeRecord.Amount),
	}
}

func issuedChequeRecordFromModel(model *chequehandler.IssuedChequeRecord) *issuedChequeRecord {
	return &issuedChequeRecord{
		ChequeRecordID: model.ChequeRecordID,
		Counter:        model.Counter.Bytes(),
		Amount:         model.Amount.Bytes(),
	}
}
