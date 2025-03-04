// Copyright (C) 2022-2025, Chain4Travel AG. All rights reserved.
// See the file LICENSE for licensing terms.

package tokenStorage

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/chain4travel/camino-messenger-bot/pkg/database/sqlite"
	"github.com/jmoiron/sqlx"
)

const tokenRecordsTableName = "tokens"

var (
	_ TokenRecordsStorage = (*storage)(nil)
)

type tokenRecordsStatements struct {
	saveTokenRecord       *sqlx.Stmt
	getTokenRecord        *sqlx.Stmt
	updateTokenRecord     *sqlx.Stmt
	getActiveTokenRecords *sqlx.Stmt
}

func (s *storage) prepareTokenRecordsStmts(ctx context.Context) error {
	saveTokenRecord, err := s.base.DB.PreparexContext(ctx, fmt.Sprintf(`
		INSERT INTO %s (token_id, created_at, expires_at, bought, expired)
		VALUES ($1, $2, $3, $4, $5)
	`, tokenRecordsTableName))
	if err != nil {
		s.base.Logger.Error(err)
		return err
	}
	s.saveTokenRecord = saveTokenRecord

	getTokenRecord, err := s.base.DB.PreparexContext(ctx, fmt.Sprintf(`
		SELECT token_id, bought, expired, created_at, expires_at
		FROM %s
		WHERE token_id = $1
	`, tokenRecordsTableName))
	if err != nil {
		s.base.Logger.Error(err)
		return err
	}
	s.getTokenRecord = getTokenRecord

	updateTokenRecord, err := s.base.DB.PreparexContext(ctx, fmt.Sprintf(`
		UPDATE %s SET bought = $1, expired = $2 WHERE token_id = $3`,
		tokenRecordsTableName))
	if err != nil {
		s.base.Logger.Error(err)
		return err
	}
	s.updateTokenRecord = updateTokenRecord

	getActiveTokenRecords, err := s.base.DB.PreparexContext(ctx, fmt.Sprintf(`
		SELECT token_id, bought, expired, created_at, expires_at
		FROM %s
		WHERE bought = false AND expired = false
	`, tokenRecordsTableName))
	if err != nil {
		s.base.Logger.Error(err)
		return err
	}
	s.getActiveTokenRecords = getActiveTokenRecords

	return nil
}

func (s *storage) SaveTokenRecord(ctx context.Context, session Session, record *TokenRecord) error {
	tx, err := getSQLXTx(session.(sqlite.Session))
	if err != nil {
		s.base.Logger.Error(err)
		return err
	}

	_, err = tx.StmtxContext(ctx, s.saveTokenRecord).ExecContext(ctx,
		record.TokenID,
		record.CreatedAt,
		record.ExpiresAt,
		record.Bought,
		record.Expired,
	)
	if err != nil {
		s.base.Logger.Error(err)
		return err
	}

	if err := session.Commit(); err != nil {
		s.base.Logger.Error(err)
		return err
	}
	return nil
}

func (s *storage) GetTokenRecord(ctx context.Context, session Session, tokenID string) (*TokenRecord, error) {
	tx, err := getSQLXTx(session.(sqlite.Session))
	if err != nil {
		s.base.Logger.Error(err)
		return nil, err
	}

	// Log the token ID we're trying to retrieve
	s.base.Logger.Infof("Attempting to retrieve token record with ID: '%s', length: %d", tokenID, len(tokenID))

	var record TokenRecord
	err = tx.StmtxContext(ctx, s.getTokenRecord).GetContext(ctx, &record, tokenID)
	if err != nil {
		if !errors.Is(err, sql.ErrNoRows) {
			s.base.Logger.Error(err)
		}
		return nil, upgradeError(err)
	}

	// Log the retrieved token ID for comparison
	s.base.Logger.Infof("Successfully retrieved token record with ID: '%s', length: %d", record.TokenID, len(record.TokenID))

	if err := session.Abort(); err != nil {
		s.base.Logger.Error(err)
		return nil, err
	}
	return &record, nil
}

func (s *storage) UpdateTokenRecord(ctx context.Context, session Session, record *TokenRecord) error {
	tx, err := getSQLXTx(session.(sqlite.Session))
	if err != nil {
		s.base.Logger.Error(err)
		return err
	}

	// Try a direct update with the raw SQL
	rawQuery := fmt.Sprintf("UPDATE %s SET bought = ?, expired = ? WHERE token_id = ?", tokenRecordsTableName)
	rawResult, err := tx.ExecContext(ctx, rawQuery, record.Bought, record.Expired, record.TokenID)
	if err != nil {
		s.base.Logger.Errorf("Raw update error: %v", err)
	} else {
		rawAffected, _ := rawResult.RowsAffected()
		s.base.Logger.Infof("Raw update affected %d rows", rawAffected)
	}

	if err != nil {
		s.base.Logger.Error(err)
		return err
	}

	// Check if any rows were actually updated
	rowsAffected, err := rawResult.RowsAffected()
	if err != nil {
		s.base.Logger.Error(fmt.Errorf("failed to get rows affected: %w", err))
		return err
	}

	if rowsAffected == 0 {
		// No rows were updated, which means the token ID wasn't found
		err := fmt.Errorf("no token record found with ID: '%s' during update", record.TokenID)
		s.base.Logger.Error(err)
		return err
	}

	if err := session.Commit(); err != nil {
		s.base.Logger.Error(err)
		return err
	}

	s.base.Logger.Infof("Successfully updated token record with ID: '%s'", record.TokenID)
	return nil
}

func (s *storage) GetActiveTokenRecords(ctx context.Context, session Session) ([]*TokenRecord, error) {
	tx, err := getSQLXTx(session.(sqlite.Session))
	if err != nil {
		s.base.Logger.Error(err)
		return nil, err
	}

	var records []*TokenRecord
	err = tx.StmtxContext(ctx, s.getActiveTokenRecords).SelectContext(ctx, &records)
	if err != nil {
		s.base.Logger.Error(err)
		return nil, err
	}

	if err := session.Abort(); err != nil {
		s.base.Logger.Error(err)
		return nil, err
	}
	return records, nil
}
