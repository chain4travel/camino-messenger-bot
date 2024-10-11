package sqlite

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"math/big"

<<<<<<<< HEAD:pkg/chequehandler/storage/sqlite/cheque_records.go
	"github.com/chain4travel/camino-messenger-bot/pkg/chequehandler"
========
	"github.com/chain4travel/camino-messenger-bot/pkg/cheque_handler"
>>>>>>>> a1cdaeb (wip):pkg/cheque_handler/storage/sqlite/cheque_records.go
	"github.com/chain4travel/camino-messenger-bot/pkg/cheques"
	"github.com/ethereum/go-ethereum/common"
	"github.com/jmoiron/sqlx"
)

const chequeRecordsTableName = "cheque_records"

var (
<<<<<<<< HEAD:pkg/chequehandler/storage/sqlite/cheque_records.go
	_ chequehandler.ChequeRecordsStorage = (*storage)(nil)
========
	_ cheque_handler.ChequeRecordsStorage = (*storage)(nil)
>>>>>>>> a1cdaeb (wip):pkg/cheque_handler/storage/sqlite/cheque_records.go

	zeroHash = common.Hash{}
)

type chequeRecord struct {
<<<<<<<< HEAD:pkg/chequehandler/storage/sqlite/cheque_records.go
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
========
	ChequeRecordID common.Hash                    `db:"cheque_record_id"`
	FromCMAccount  common.Address                 `db:"from_cm_account"`
	ToCMAccount    common.Address                 `db:"to_cm_account"`
	ToBot          common.Address                 `db:"to_bot"`
	Counter        []byte                         `db:"counter"`
	Amount         []byte                         `db:"amount"`
	CreatedAt      []byte                         `db:"created_at"`
	ExpiresAt      []byte                         `db:"expires_at"`
	Signature      []byte                         `db:"signature"`
	TxID           *common.Hash                   `db:"tx_id"`
	Status         *cheque_handler.ChequeTxStatus `db:"status"`
}

func (s *storage) GetNotCashedChequeRecords(ctx context.Context, session cheque_handler.Session) ([]*cheque_handler.ChequeRecord, error) {
	tx, err := getSQLXTx(session)
	if err != nil {
		s.baseDB.Logger.Error(err)
		return nil, err
	}

	chequeRecords := []*cheque_handler.ChequeRecord{}
	rows, err := tx.StmtxContext(ctx, s.getNotCashedChequeRecords).QueryxContext(ctx)
	if err != nil {
		s.baseDB.Logger.Error(err)
>>>>>>>> a1cdaeb (wip):pkg/cheque_handler/storage/sqlite/cheque_records.go
		return nil, upgradeError(err)
	}
	for rows.Next() {
		chequeRecord := &chequeRecord{}
		if err := rows.StructScan(chequeRecord); err != nil {
<<<<<<<< HEAD:pkg/chequehandler/storage/sqlite/cheque_records.go
			s.base.Logger.Errorf("failed to get not cashed chequeRecord from db: %v", err)
========
			s.baseDB.Logger.Errorf("failed to get not cashed chequeRecord from db: %v", err)
>>>>>>>> a1cdaeb (wip):pkg/cheque_handler/storage/sqlite/cheque_records.go
			continue
		}
		model, err := modelFromChequeRecord(chequeRecord)
		if err != nil {
<<<<<<<< HEAD:pkg/chequehandler/storage/sqlite/cheque_records.go
			s.base.Logger.Errorf("failed to parse not cashed chequeRecord: %v", err)
========
			s.baseDB.Logger.Errorf("failed to parse not cashed chequeRecord: %v", err)
>>>>>>>> a1cdaeb (wip):pkg/cheque_handler/storage/sqlite/cheque_records.go
			continue
		}
		chequeRecords = append(chequeRecords, model)
	}
	return chequeRecords, nil
}

<<<<<<<< HEAD:pkg/chequehandler/storage/sqlite/cheque_records.go
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
========
func (s *storage) GetChequeRecordsWithPendingTxs(ctx context.Context, session cheque_handler.Session) ([]*cheque_handler.ChequeRecord, error) {
	tx, err := getSQLXTx(session)
	if err != nil {
		s.baseDB.Logger.Error(err)
		return nil, err
	}

	chequeRecords := []*cheque_handler.ChequeRecord{}
	rows, err := tx.StmtxContext(ctx, s.getChequeRecordsWithPendingTxs).QueryxContext(ctx)
	if err != nil {
		s.baseDB.Logger.Error(err)
>>>>>>>> a1cdaeb (wip):pkg/cheque_handler/storage/sqlite/cheque_records.go
		return nil, upgradeError(err)
	}
	for rows.Next() {
		chequeRecord := &chequeRecord{}
		if err := rows.StructScan(chequeRecord); err != nil {
<<<<<<<< HEAD:pkg/chequehandler/storage/sqlite/cheque_records.go
			s.base.Logger.Errorf("failed to get chequeRecord with pending tx from db: %v", err)
========
			s.baseDB.Logger.Errorf("failed to get chequeRecord with pending tx from db: %v", err)
>>>>>>>> a1cdaeb (wip):pkg/cheque_handler/storage/sqlite/cheque_records.go
			continue
		}
		model, err := modelFromChequeRecord(chequeRecord)
		if err != nil {
<<<<<<<< HEAD:pkg/chequehandler/storage/sqlite/cheque_records.go
			s.base.Logger.Errorf("failed to parse chequeRecord with pending tx: %v", err)
========
			s.baseDB.Logger.Errorf("failed to parse chequeRecord with pending tx: %v", err)
>>>>>>>> a1cdaeb (wip):pkg/cheque_handler/storage/sqlite/cheque_records.go
			continue
		}
		chequeRecords = append(chequeRecords, model)
	}
	return chequeRecords, nil
}

<<<<<<<< HEAD:pkg/chequehandler/storage/sqlite/cheque_records.go
func (s *storage) GetChequeRecord(ctx context.Context, session chequehandler.Session, chequeRecordID common.Hash) (*chequehandler.ChequeRecord, error) {
	tx, err := getSQLXTx(session)
	if err != nil {
		s.base.Logger.Error(err)
========
func (s *storage) GetChequeRecord(ctx context.Context, session cheque_handler.Session, chequeRecordID common.Hash) (*cheque_handler.ChequeRecord, error) {
	tx, err := getSQLXTx(session)
	if err != nil {
		s.baseDB.Logger.Error(err)
>>>>>>>> a1cdaeb (wip):pkg/cheque_handler/storage/sqlite/cheque_records.go
		return nil, err
	}

	chequeRecord := &chequeRecord{}
	if err := tx.StmtxContext(ctx, s.getChequeRecordByID).GetContext(ctx, chequeRecord, chequeRecordID); err != nil {
		if !errors.Is(err, sql.ErrNoRows) {
<<<<<<<< HEAD:pkg/chequehandler/storage/sqlite/cheque_records.go
			s.base.Logger.Error(err)
========
			s.baseDB.Logger.Error(err)
>>>>>>>> a1cdaeb (wip):pkg/cheque_handler/storage/sqlite/cheque_records.go
		}
		return nil, upgradeError(err)
	}
	return modelFromChequeRecord(chequeRecord)
}

<<<<<<<< HEAD:pkg/chequehandler/storage/sqlite/cheque_records.go
func (s *storage) GetChequeRecordByTxID(ctx context.Context, session chequehandler.Session, txID common.Hash) (*chequehandler.ChequeRecord, error) {
	tx, err := getSQLXTx(session)
	if err != nil {
		s.base.Logger.Error(err)
========
func (s *storage) GetChequeRecordByTxID(ctx context.Context, session cheque_handler.Session, txID common.Hash) (*cheque_handler.ChequeRecord, error) {
	tx, err := getSQLXTx(session)
	if err != nil {
		s.baseDB.Logger.Error(err)
>>>>>>>> a1cdaeb (wip):pkg/cheque_handler/storage/sqlite/cheque_records.go
		return nil, err
	}

	chequeRecord := &chequeRecord{}
	if err := tx.StmtxContext(ctx, s.getChequeRecordByTxID).GetContext(ctx, chequeRecord, txID); err != nil {
		if !errors.Is(err, sql.ErrNoRows) {
<<<<<<<< HEAD:pkg/chequehandler/storage/sqlite/cheque_records.go
			s.base.Logger.Error(err)
========
			s.baseDB.Logger.Error(err)
>>>>>>>> a1cdaeb (wip):pkg/cheque_handler/storage/sqlite/cheque_records.go
		}
		return nil, upgradeError(err)
	}
	return modelFromChequeRecord(chequeRecord)
}

<<<<<<<< HEAD:pkg/chequehandler/storage/sqlite/cheque_records.go
func (s *storage) UpsertChequeRecord(ctx context.Context, session chequehandler.Session, chequeRecord *chequehandler.ChequeRecord) error {
	tx, err := getSQLXTx(session)
	if err != nil {
		s.base.Logger.Error(err)
========
func (s *storage) UpsertChequeRecord(ctx context.Context, session cheque_handler.Session, chequeRecord *cheque_handler.ChequeRecord) error {
	tx, err := getSQLXTx(session)
	if err != nil {
		s.baseDB.Logger.Error(err)
>>>>>>>> a1cdaeb (wip):pkg/cheque_handler/storage/sqlite/cheque_records.go
		return err
	}

	result, err := tx.NamedStmtContext(ctx, s.upsertChequeRecord).
		ExecContext(ctx, chequeRecordFromModel(chequeRecord))
	if err != nil {
<<<<<<<< HEAD:pkg/chequehandler/storage/sqlite/cheque_records.go
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
>>>>>>>> a1cdaeb (wip):pkg/cheque_handler/storage/sqlite/cheque_records.go
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
<<<<<<<< HEAD:pkg/chequehandler/storage/sqlite/cheque_records.go
	getNotCashedChequeRecords, err := s.base.DB.PreparexContext(ctx, fmt.Sprintf(`
		SELECT * FROM %s
		WHERE status = %d OR status IS NULL
	`, chequeRecordsTableName, chequehandler.ChequeTxStatusRejected))
	if err != nil {
		s.base.Logger.Error(err)
========
	getNotCashedChequeRecords, err := s.baseDB.DB.PreparexContext(ctx, fmt.Sprintf(`
		SELECT * FROM %s
		WHERE status = %d OR status IS NULL
	`, chequeRecordsTableName, cheque_handler.ChequeTxStatusRejected))
	if err != nil {
		s.baseDB.Logger.Error(err)
>>>>>>>> a1cdaeb (wip):pkg/cheque_handler/storage/sqlite/cheque_records.go
		return err
	}
	s.getNotCashedChequeRecords = getNotCashedChequeRecords

<<<<<<<< HEAD:pkg/chequehandler/storage/sqlite/cheque_records.go
	getChequeRecordsWithPendingTxs, err := s.base.DB.PreparexContext(ctx, fmt.Sprintf(`
		SELECT * FROM %s
		WHERE status = %d
	`, chequeRecordsTableName, chequehandler.ChequeTxStatusPending))
	if err != nil {
		s.base.Logger.Error(err)
========
	getChequeRecordsWithPendingTxs, err := s.baseDB.DB.PreparexContext(ctx, fmt.Sprintf(`
		SELECT * FROM %s
		WHERE status = %d
	`, chequeRecordsTableName, cheque_handler.ChequeTxStatusPending))
	if err != nil {
		s.baseDB.Logger.Error(err)
>>>>>>>> a1cdaeb (wip):pkg/cheque_handler/storage/sqlite/cheque_records.go
		return err
	}
	s.getChequeRecordsWithPendingTxs = getChequeRecordsWithPendingTxs

<<<<<<<< HEAD:pkg/chequehandler/storage/sqlite/cheque_records.go
	getChequeRecordByID, err := s.base.DB.PreparexContext(ctx, fmt.Sprintf(`
========
	getChequeRecordByID, err := s.baseDB.DB.PreparexContext(ctx, fmt.Sprintf(`
>>>>>>>> a1cdaeb (wip):pkg/cheque_handler/storage/sqlite/cheque_records.go
		SELECT * FROM %s
		WHERE cheque_record_id = ?
	`, chequeRecordsTableName))
	if err != nil {
<<<<<<<< HEAD:pkg/chequehandler/storage/sqlite/cheque_records.go
		s.base.Logger.Error(err)
========
		s.baseDB.Logger.Error(err)
>>>>>>>> a1cdaeb (wip):pkg/cheque_handler/storage/sqlite/cheque_records.go
		return err
	}
	s.getChequeRecordByID = getChequeRecordByID

<<<<<<<< HEAD:pkg/chequehandler/storage/sqlite/cheque_records.go
	getChequeByTxID, err := s.base.DB.PreparexContext(ctx, fmt.Sprintf(`
========
	getChequeByTxID, err := s.baseDB.DB.PreparexContext(ctx, fmt.Sprintf(`
>>>>>>>> a1cdaeb (wip):pkg/cheque_handler/storage/sqlite/cheque_records.go
		SELECT * FROM %s
		WHERE tx_id = ?
	`, chequeRecordsTableName))
	if err != nil {
<<<<<<<< HEAD:pkg/chequehandler/storage/sqlite/cheque_records.go
		s.base.Logger.Error(err)
========
		s.baseDB.Logger.Error(err)
>>>>>>>> a1cdaeb (wip):pkg/cheque_handler/storage/sqlite/cheque_records.go
		return err
	}
	s.getChequeRecordByTxID = getChequeByTxID

<<<<<<<< HEAD:pkg/chequehandler/storage/sqlite/cheque_records.go
	upsertChequeRecord, err := s.base.DB.PrepareNamedContext(ctx, fmt.Sprintf(`
========
	upsertChequeRecord, err := s.baseDB.DB.PrepareNamedContext(ctx, fmt.Sprintf(`
>>>>>>>> a1cdaeb (wip):pkg/cheque_handler/storage/sqlite/cheque_records.go
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
<<<<<<<< HEAD:pkg/chequehandler/storage/sqlite/cheque_records.go
		s.base.Logger.Error(err)
========
		s.baseDB.Logger.Error(err)
>>>>>>>> a1cdaeb (wip):pkg/cheque_handler/storage/sqlite/cheque_records.go
		return err
	}
	s.upsertChequeRecord = upsertChequeRecord

	return nil
}

<<<<<<<< HEAD:pkg/chequehandler/storage/sqlite/cheque_records.go
func modelFromChequeRecord(chequeRecord *chequeRecord) (*chequehandler.ChequeRecord, error) {
========
func modelFromChequeRecord(chequeRecord *chequeRecord) (*cheque_handler.ChequeRecord, error) {
>>>>>>>> a1cdaeb (wip):pkg/cheque_handler/storage/sqlite/cheque_records.go
	txID := common.Hash{}
	if chequeRecord.TxID != nil {
		txID = *chequeRecord.TxID
	}

<<<<<<<< HEAD:pkg/chequehandler/storage/sqlite/cheque_records.go
	status := chequehandler.ChequeTxStatusUnknown
========
	status := cheque_handler.ChequeTxStatusUnknown
>>>>>>>> a1cdaeb (wip):pkg/cheque_handler/storage/sqlite/cheque_records.go
	if chequeRecord.Status != nil {
		status = *chequeRecord.Status
	}

<<<<<<<< HEAD:pkg/chequehandler/storage/sqlite/cheque_records.go
	return &chequehandler.ChequeRecord{
========
	return &cheque_handler.ChequeRecord{
>>>>>>>> a1cdaeb (wip):pkg/cheque_handler/storage/sqlite/cheque_records.go
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

<<<<<<<< HEAD:pkg/chequehandler/storage/sqlite/cheque_records.go
func chequeRecordFromModel(model *chequehandler.ChequeRecord) *chequeRecord {
========
func chequeRecordFromModel(model *cheque_handler.ChequeRecord) *chequeRecord {
>>>>>>>> a1cdaeb (wip):pkg/cheque_handler/storage/sqlite/cheque_records.go
	var txID *common.Hash
	if model.TxID != zeroHash {
		txID = &model.TxID
	}

<<<<<<<< HEAD:pkg/chequehandler/storage/sqlite/cheque_records.go
	var status *chequehandler.ChequeTxStatus
	if model.Status != chequehandler.ChequeTxStatusUnknown {
========
	var status *cheque_handler.ChequeTxStatus
	if model.Status != cheque_handler.ChequeTxStatusUnknown {
>>>>>>>> a1cdaeb (wip):pkg/cheque_handler/storage/sqlite/cheque_records.go
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
