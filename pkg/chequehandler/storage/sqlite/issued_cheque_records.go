package sqlite

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"math/big"

<<<<<<<< HEAD:pkg/chequehandler/storage/sqlite/issued_cheque_records.go
	"github.com/chain4travel/camino-messenger-bot/pkg/chequehandler"
========
	"github.com/chain4travel/camino-messenger-bot/pkg/cheque_handler"
>>>>>>>> a1cdaeb (wip):pkg/cheque_handler/storage/sqlite/issued_cheque_records.go
	"github.com/ethereum/go-ethereum/common"
	"github.com/jmoiron/sqlx"
)

const issuedChequeRecordsTableName = "issued_cheque_records"

<<<<<<<< HEAD:pkg/chequehandler/storage/sqlite/issued_cheque_records.go
var _ chequehandler.ChequeRecordsStorage = (*storage)(nil)
========
var _ cheque_handler.ChequeRecordsStorage = (*storage)(nil)
>>>>>>>> a1cdaeb (wip):pkg/cheque_handler/storage/sqlite/issued_cheque_records.go

type issuedChequeRecord struct {
	ChequeRecordID common.Hash `db:"cheque_record_id"`
	Counter        []byte      `db:"counter"`
	Amount         []byte      `db:"amount"`
}

<<<<<<<< HEAD:pkg/chequehandler/storage/sqlite/issued_cheque_records.go
func (s *storage) GetIssuedChequeRecord(ctx context.Context, session chequehandler.Session, chequeRecordID common.Hash) (*chequehandler.IssuedChequeRecord, error) {
	tx, err := getSQLXTx(session)
	if err != nil {
		s.base.Logger.Error(err)
========
func (s *storage) GetIssuedChequeRecord(ctx context.Context, session cheque_handler.Session, chequeRecordID common.Hash) (*cheque_handler.IssuedChequeRecord, error) {
	tx, err := getSQLXTx(session)
	if err != nil {
		s.baseDB.Logger.Error(err)
>>>>>>>> a1cdaeb (wip):pkg/cheque_handler/storage/sqlite/issued_cheque_records.go
		return nil, err
	}

	chequeRecord := &issuedChequeRecord{}
	if err := tx.StmtxContext(ctx, s.getIssuedChequeRecord).GetContext(ctx, chequeRecord, chequeRecordID); err != nil {
		if !errors.Is(err, sql.ErrNoRows) {
<<<<<<<< HEAD:pkg/chequehandler/storage/sqlite/issued_cheque_records.go
			s.base.Logger.Error(err)
========
			s.baseDB.Logger.Error(err)
>>>>>>>> a1cdaeb (wip):pkg/cheque_handler/storage/sqlite/issued_cheque_records.go
		}
		return nil, upgradeError(err)
	}
	return modelFromIssuedChequeRecord(chequeRecord), nil
}

<<<<<<<< HEAD:pkg/chequehandler/storage/sqlite/issued_cheque_records.go
func (s *storage) UpsertIssuedChequeRecord(ctx context.Context, session chequehandler.Session, chequeRecord *chequehandler.IssuedChequeRecord) error {
	tx, err := getSQLXTx(session)
	if err != nil {
		s.base.Logger.Error(err)
========
func (s *storage) UpsertIssuedChequeRecord(ctx context.Context, session cheque_handler.Session, chequeRecord *cheque_handler.IssuedChequeRecord) error {
	tx, err := getSQLXTx(session)
	if err != nil {
		s.baseDB.Logger.Error(err)
>>>>>>>> a1cdaeb (wip):pkg/cheque_handler/storage/sqlite/issued_cheque_records.go
		return err
	}

	result, err := tx.NamedStmtContext(ctx, s.upsertIssuedChequeRecord).
		ExecContext(ctx, issuedChequeRecordFromModel(chequeRecord))
	if err != nil {
<<<<<<<< HEAD:pkg/chequehandler/storage/sqlite/issued_cheque_records.go
		s.base.Logger.Error(err)
		return upgradeError(err)
	}
	if rowsAffected, err := result.RowsAffected(); err != nil {
		s.base.Logger.Error(err)
========
		s.baseDB.Logger.Error(err)
		return upgradeError(err)
	}
	if rowsAffected, err := result.RowsAffected(); err != nil {
		s.baseDB.Logger.Error(err)
>>>>>>>> a1cdaeb (wip):pkg/cheque_handler/storage/sqlite/issued_cheque_records.go
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
<<<<<<<< HEAD:pkg/chequehandler/storage/sqlite/issued_cheque_records.go
	getIssuedChequeRecord, err := s.base.DB.PreparexContext(ctx, fmt.Sprintf(`
========
	getIssuedChequeRecord, err := s.baseDB.DB.PreparexContext(ctx, fmt.Sprintf(`
>>>>>>>> a1cdaeb (wip):pkg/cheque_handler/storage/sqlite/issued_cheque_records.go
		SELECT * FROM %s
		WHERE cheque_record_id = ?
	`, issuedChequeRecordsTableName))
	if err != nil {
<<<<<<<< HEAD:pkg/chequehandler/storage/sqlite/issued_cheque_records.go
		s.base.Logger.Error(err)
========
		s.baseDB.Logger.Error(err)
>>>>>>>> a1cdaeb (wip):pkg/cheque_handler/storage/sqlite/issued_cheque_records.go
		return err
	}
	s.getIssuedChequeRecord = getIssuedChequeRecord

<<<<<<<< HEAD:pkg/chequehandler/storage/sqlite/issued_cheque_records.go
	upsertIssuedChequeRecord, err := s.base.DB.PrepareNamedContext(ctx, fmt.Sprintf(`
========
	upsertIssuedChequeRecord, err := s.baseDB.DB.PrepareNamedContext(ctx, fmt.Sprintf(`
>>>>>>>> a1cdaeb (wip):pkg/cheque_handler/storage/sqlite/issued_cheque_records.go
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
<<<<<<<< HEAD:pkg/chequehandler/storage/sqlite/issued_cheque_records.go
		s.base.Logger.Error(err)
========
		s.baseDB.Logger.Error(err)
>>>>>>>> a1cdaeb (wip):pkg/cheque_handler/storage/sqlite/issued_cheque_records.go
		return err
	}
	s.upsertIssuedChequeRecord = upsertIssuedChequeRecord

	return nil
}

<<<<<<<< HEAD:pkg/chequehandler/storage/sqlite/issued_cheque_records.go
func modelFromIssuedChequeRecord(chequeRecord *issuedChequeRecord) *chequehandler.IssuedChequeRecord {
	return &chequehandler.IssuedChequeRecord{
========
func modelFromIssuedChequeRecord(chequeRecord *issuedChequeRecord) *cheque_handler.IssuedChequeRecord {
	return &cheque_handler.IssuedChequeRecord{
>>>>>>>> a1cdaeb (wip):pkg/cheque_handler/storage/sqlite/issued_cheque_records.go
		ChequeRecordID: chequeRecord.ChequeRecordID,
		Counter:        big.NewInt(0).SetBytes(chequeRecord.Counter),
		Amount:         big.NewInt(0).SetBytes(chequeRecord.Amount),
	}
}

<<<<<<<< HEAD:pkg/chequehandler/storage/sqlite/issued_cheque_records.go
func issuedChequeRecordFromModel(model *chequehandler.IssuedChequeRecord) *issuedChequeRecord {
========
func issuedChequeRecordFromModel(model *cheque_handler.IssuedChequeRecord) *issuedChequeRecord {
>>>>>>>> a1cdaeb (wip):pkg/cheque_handler/storage/sqlite/issued_cheque_records.go
	return &issuedChequeRecord{
		ChequeRecordID: model.ChequeRecordID,
		Counter:        model.Counter.Bytes(),
		Amount:         model.Amount.Bytes(),
	}
}
