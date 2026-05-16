// Copyright (C) 2022-2026, Chain4Travel AG. All rights reserved.
// See the file LICENSE for licensing terms.

package sqlite

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"math/big"

	"github.com/chain4travel/camino-messenger-bot/v13/internal/eventlistener"
	"github.com/chain4travel/camino-messenger-bot/v13/pkg/database/sqlite"
	"github.com/jmoiron/sqlx"
)

const cancellationSubscriptionsTable = "cancellation_subscriptions"

var _ eventlistener.Storage = (*storage)(nil)

func (s *storage) AddCancellationSubscription(ctx context.Context, session eventlistener.Session, tokenID *big.Int) error {
	tx, err := sqlite.GetSQLXTx(session)
	if err != nil {
		return fmt.Errorf("failed to get transaction from db session: %w", err)
	}

	result, err := tx.StmtxContext(ctx, s.insertCancellationSubscription).ExecContext(ctx, tokenID.Int64())
	if err != nil {
		return fmt.Errorf("failed to execute insert cancellation subscription statement: %w", err)
	}
	if rowsAffected, err := result.RowsAffected(); err != nil {
		return fmt.Errorf("failed to get rowsAffected from statement execution result: %w", err)
	} else if rowsAffected != 1 {
		return fmt.Errorf("unexpected number of rows affected: expected 1, but affected %d", rowsAffected)
	}
	return nil
}

func (s *storage) RemoveCancellationSubscription(ctx context.Context, session eventlistener.Session, tokenID *big.Int) error {
	tx, err := sqlite.GetSQLXTx(session)
	if err != nil {
		return fmt.Errorf("failed to get transaction from db session: %w", err)
	}

	result, err := tx.StmtxContext(ctx, s.removeCancellationSubscription).ExecContext(ctx, tokenID.Int64())
	if err != nil {
		return fmt.Errorf("failed to execute remove cancellation subscription statement: %w", err)
	}
	if rowsAffected, err := result.RowsAffected(); err != nil {
		return fmt.Errorf("failed to get rowsAffected from statement execution result: %w", err)
	} else if rowsAffected != 1 {
		return fmt.Errorf("unexpected number of rows affected: expected 1, but affected %d", rowsAffected)
	}
	return nil
}

func (s *storage) GetAllCancellationSubscriptions(ctx context.Context, session eventlistener.Session) ([]*big.Int, error) {
	tx, err := sqlite.GetSQLXTx(session)
	if err != nil {
		return nil, fmt.Errorf("failed to get transaction from db session: %w", err)
	}

	subscriptions := []*big.Int{}
	rows, err := tx.StmtxContext(ctx, s.getAllCancellationSubscription).QueryxContext(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to execute get all cancellation subscriptions statement: %w", err)
	}
	for rows.Next() {
		tokenID := int64(0)
		if err := rows.Scan(&tokenID); err != nil {
			return nil, fmt.Errorf("failed to scan row to tokenID: %w", err)
		}
		subscriptions = append(subscriptions, big.NewInt(tokenID))
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error occurred during rows iteration: %w", err)
	}
	return subscriptions, nil
}

func (s *storage) IsCancellationSubscriptionExist(ctx context.Context, session eventlistener.Session, tokenID *big.Int) (bool, error) {
	tx, err := sqlite.GetSQLXTx(session)
	if err != nil {
		return false, fmt.Errorf("failed to get transaction from db session: %w", err)
	}

	dbTokenID := int64(0)
	if err := tx.StmtxContext(ctx, s.getCancellationSubscription).GetContext(ctx, &dbTokenID, tokenID.Int64()); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return false, nil
		}
		return false, fmt.Errorf("failed to execute get cancellation subscription statement: %w", err)
	}
	return true, nil
}

type cancellationSubscriptionsStatements struct {
	insertCancellationSubscription *sqlx.Stmt
	removeCancellationSubscription *sqlx.Stmt
	getAllCancellationSubscription *sqlx.Stmt
	getCancellationSubscription    *sqlx.Stmt
}

func (s *storage) prepareCancellationSubscriptionsStmts(ctx context.Context) error {
	insertCancellationSubscription, err := s.base.DB.PreparexContext(ctx, fmt.Sprintf(`
		INSERT INTO %s ( token_id ) VALUES ( ? )
	`, cancellationSubscriptionsTable))
	if err != nil {
		return fmt.Errorf("failed to prepare insert cancellation subscription statement: %w", err)
	}
	s.insertCancellationSubscription = insertCancellationSubscription

	removeCancellationSubscription, err := s.base.DB.PreparexContext(ctx, fmt.Sprintf(`
		DELETE FROM %s
		WHERE token_id = ?
	`, cancellationSubscriptionsTable))
	if err != nil {
		return fmt.Errorf("failed to prepare remove cancellation subscription statement: %w", err)
	}
	s.removeCancellationSubscription = removeCancellationSubscription

	getAllCancellationSubscription, err := s.base.DB.PreparexContext(ctx, fmt.Sprintf(`
		SELECT * FROM %s
	`, cancellationSubscriptionsTable))
	if err != nil {
		return fmt.Errorf("failed to prepare get all cancellation subscriptions statement: %w", err)
	}
	s.getAllCancellationSubscription = getAllCancellationSubscription

	getCancellationSubscription, err := s.base.DB.PreparexContext(ctx, fmt.Sprintf(`
		SELECT * FROM %s
		WHERE token_id = ?
	`, cancellationSubscriptionsTable))
	if err != nil {
		return fmt.Errorf("failed to prepare get cancellation subscription statement: %w", err)
	}
	s.getCancellationSubscription = getCancellationSubscription

	return nil
}
