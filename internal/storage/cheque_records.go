package storage

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"math/big"

	"github.com/chain4travel/camino-messenger-bot/internal/models"
	"github.com/chain4travel/camino-messenger-bot/pkg/cheques"
	"github.com/ethereum/go-ethereum/common"
	"github.com/jmoiron/sqlx"
)

const chequeRecordsTableName = "cheque_records"

var (
	_ ChequeRecordsStorage = (*session)(nil)

	zeroHash = common.Hash{}
)

type ChequeRecordsStorage interface {
	GetNotCashedChequeRecords(ctx context.Context) ([]*models.ChequeRecord, error)
	GetChequeRecordsWithPendingTxs(ctx context.Context) ([]*models.ChequeRecord, error)
	GetChequeRecord(ctx context.Context, chequeRecordID common.Hash) (*models.ChequeRecord, error)
	GetChequeRecordByTxID(ctx context.Context, txID common.Hash) (*models.ChequeRecord, error)
	UpsertChequeRecord(ctx context.Context, chequeRecord *models.ChequeRecord) error
}

type chequeRecord struct {
	ChequeRecordID common.Hash            `db:"cheque_record_id"`
	FromCMAccount  common.Address         `db:"from_cm_account"`
	ToCMAccount    common.Address         `db:"to_cm_account"`
	ToBot          common.Address         `db:"to_bot"`
	Counter        []byte                 `db:"counter"`
	Amount         []byte                 `db:"amount"`
	CreatedAt      []byte                 `db:"created_at"`
	ExpiresAt      []byte                 `db:"expires_at"`
	Signature      []byte                 `db:"signature"`
	TxID           *common.Hash           `db:"tx_id"`
	Status         *models.ChequeTxStatus `db:"status"`
}

func (s *session) GetNotCashedChequeRecords(ctx context.Context) ([]*models.ChequeRecord, error) {
	chequeRecords := []*models.ChequeRecord{}
	rows, err := s.tx.StmtxContext(ctx, s.storage.getNotCashedChequeRecords).QueryxContext(ctx)
	if err != nil {
		s.logger.Error(err)
		return nil, upgradeError(err)
	}
	for rows.Next() {
		chequeRecord := &chequeRecord{}
		if err := rows.StructScan(chequeRecord); err != nil {
			s.logger.Errorf("failed to get not cashed chequeRecord from db: %v", err)
			continue
		}
		model, err := modelFromChequeRecord(chequeRecord)
		if err != nil {
			s.logger.Errorf("failed to parse not cashed chequeRecord: %v", err)
			continue
		}
		chequeRecords = append(chequeRecords, model)
	}
	return chequeRecords, nil
}

func (s *session) GetChequeRecordsWithPendingTxs(ctx context.Context) ([]*models.ChequeRecord, error) {
	chequeRecords := []*models.ChequeRecord{}
	rows, err := s.tx.StmtxContext(ctx, s.storage.getChequeRecordsWithPendingTxs).QueryxContext(ctx)
	if err != nil {
		s.logger.Error(err)
		return nil, upgradeError(err)
	}
	for rows.Next() {
		chequeRecord := &chequeRecord{}
		if err := rows.StructScan(chequeRecord); err != nil {
			s.logger.Errorf("failed to get chequeRecord with pending tx from db: %v", err)
			continue
		}
		model, err := modelFromChequeRecord(chequeRecord)
		if err != nil {
			s.logger.Errorf("failed to parse chequeRecord with pending tx: %v", err)
			continue
		}
		chequeRecords = append(chequeRecords, model)
	}
	return chequeRecords, nil
}

func (s *session) GetChequeRecord(ctx context.Context, chequeRecordID common.Hash) (*models.ChequeRecord, error) {
	chequeRecord := &chequeRecord{}
	if err := s.tx.StmtxContext(ctx, s.storage.getChequeRecordByID).GetContext(ctx, chequeRecord, chequeRecordID); err != nil {
		if !errors.Is(err, sql.ErrNoRows) {
			s.logger.Error(err)
		}
		return nil, upgradeError(err)
	}
	return modelFromChequeRecord(chequeRecord)
}

func (s *session) GetChequeRecordByTxID(ctx context.Context, txID common.Hash) (*models.ChequeRecord, error) {
	chequeRecord := &chequeRecord{}
	if err := s.tx.StmtxContext(ctx, s.storage.getChequeRecordByTxID).GetContext(ctx, chequeRecord, txID); err != nil {
		if !errors.Is(err, sql.ErrNoRows) {
			s.logger.Error(err)
		}
		return nil, upgradeError(err)
	}
	return modelFromChequeRecord(chequeRecord)
}

func (s *session) UpsertChequeRecord(ctx context.Context, chequeRecord *models.ChequeRecord) error {
	result, err := s.tx.NamedStmtContext(ctx, s.storage.upsertChequeRecord).
		ExecContext(ctx, chequeRecordFromModel(chequeRecord))
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

type chequeRecordsStatements struct {
	getNotCashedChequeRecords, getChequeRecordsWithPendingTxs *sqlx.Stmt
	getChequeRecordByID, getChequeRecordByTxID                *sqlx.Stmt
	upsertChequeRecord                                        *sqlx.NamedStmt
}

func (s *storage) prepareChequeRecordsStmts(ctx context.Context) error {
	getNotCashedChequeRecords, err := s.db.PreparexContext(ctx, fmt.Sprintf(`
		SELECT * FROM %s
		WHERE status = %d OR status IS NULL
	`, chequeRecordsTableName, models.ChequeTxStatusRejected))
	if err != nil {
		s.logger.Error(err)
		return err
	}
	s.getNotCashedChequeRecords = getNotCashedChequeRecords

	getChequeRecordsWithPendingTxs, err := s.db.PreparexContext(ctx, fmt.Sprintf(`
		SELECT * FROM %s
		WHERE status = %d
	`, chequeRecordsTableName, models.ChequeTxStatusPending))
	if err != nil {
		s.logger.Error(err)
		return err
	}
	s.getChequeRecordsWithPendingTxs = getChequeRecordsWithPendingTxs

	getChequeRecordByID, err := s.db.PreparexContext(ctx, fmt.Sprintf(`
		SELECT * FROM %s
		WHERE cheque_record_id = ?
	`, chequeRecordsTableName))
	if err != nil {
		s.logger.Error(err)
		return err
	}
	s.getChequeRecordByID = getChequeRecordByID

	getChequeByTxID, err := s.db.PreparexContext(ctx, fmt.Sprintf(`
		SELECT * FROM %s
		WHERE tx_id = ?
	`, chequeRecordsTableName))
	if err != nil {
		s.logger.Error(err)
		return err
	}
	s.getChequeRecordByTxID = getChequeByTxID

	upsertChequeRecord, err := s.db.PrepareNamedContext(ctx, fmt.Sprintf(`
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
		s.logger.Error(err)
		return err
	}
	s.upsertChequeRecord = upsertChequeRecord

	return nil
}

func modelFromChequeRecord(chequeRecord *chequeRecord) (*models.ChequeRecord, error) {
	txID := common.Hash{}
	if chequeRecord.TxID != nil {
		txID = *chequeRecord.TxID
	}

	status := models.ChequeTxStatusUnknown
	if chequeRecord.Status != nil {
		status = *chequeRecord.Status
	}

	return &models.ChequeRecord{
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

func chequeRecordFromModel(model *models.ChequeRecord) *chequeRecord {
	var txID *common.Hash
	if model.TxID != zeroHash {
		txID = &model.TxID
	}

	var status *models.ChequeTxStatus
	if model.Status != models.ChequeTxStatusUnknown {
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
