// Copyright (C) 2022-2025, Chain4Travel AG. All rights reserved.
// See the file LICENSE for licensing terms.

package sqlite

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"math/big"

	"github.com/chain4travel/camino-messenger-bot/v12/internal/eventlistener"
	"github.com/chain4travel/camino-messenger-bot/v12/pkg/database/sqlite"
	"github.com/jmoiron/sqlx"
)

const cancellationSubscriptionsTable = "cancellation_subscriptions"

var _ eventlistener.Storage = (*storage)(nil)

func (s *storage) AddCancellationSubscription(ctx context.Context, session eventlistener.Session, tokenID *big.Int) error {
	tx, err := sqlite.GetSQLXTx(session)
	if err != nil {
		s.base.Logger.Error(err)
		return err
	}

	result, err := tx.StmtxContext(ctx, s.insertCancellationSubscription).
		ExecContext(ctx, tokenID.Int64())
	if err != nil {
		err = fmt.Errorf("failed to execute insert cancellation subscription statement: %w", err)
		s.base.Logger.Error(err)
		return upgradeError(err)
	}
	if rowsAffected, err := result.RowsAffected(); err != nil {
		err = fmt.Errorf("failed to get rowsAffected from statement execution result: %w", err)
		s.base.Logger.Error(err)
		return upgradeError(err)
	} else if rowsAffected != 1 {
		err = fmt.Errorf("unexpected number of rows affected: expected 1, but affected %d", rowsAffected)
		s.base.Logger.Error(err)
		return err
	}
	return nil
}

func (s *storage) RemoveCancellationSubscription(ctx context.Context, session eventlistener.Session, tokenID *big.Int) error {
	tx, err := sqlite.GetSQLXTx(session)
	if err != nil {
		err = fmt.Errorf("failed to get transaction from session: %w", err)
		s.base.Logger.Error(err)
		return err
	}

	result, err := tx.StmtxContext(ctx, s.removeCancellationSubscription).ExecContext(ctx, tokenID.Int64())
	if err != nil {
		err = fmt.Errorf("failed to execute remove cancellation subscription statement: %w", err)
		s.base.Logger.Error(err)
		return upgradeError(err)
	}
	if rowsAffected, err := result.RowsAffected(); err != nil {
		err = fmt.Errorf("failed to get rowsAffected from statement execution result: %w", err)
		s.base.Logger.Error(err)
		return upgradeError(err)
	} else if rowsAffected != 1 {
		err = fmt.Errorf("unexpected number of rows affected: expected 1, but affected %d", rowsAffected)
		s.base.Logger.Error(err)
		return err
	}
	return nil
}

func (s *storage) GetAllCancellationSubscriptions(ctx context.Context, session eventlistener.Session) ([]*big.Int, error) {
	tx, err := sqlite.GetSQLXTx(session)
	if err != nil {
		err = fmt.Errorf("failed to get transaction from session: %w", err)
		s.base.Logger.Error(err)
		return nil, err
	}

	subscriptions := []*big.Int{}
	rows, err := tx.StmtxContext(ctx, s.getAllCancellationSubscription).QueryxContext(ctx)
	if err != nil {
		err = fmt.Errorf("failed to execute get all cancellation subscriptions statement: %w", err)
		s.base.Logger.Error(err)
		return nil, upgradeError(err)
	}
	for rows.Next() {
		tokenID := int64(0)
		if err := rows.Scan(&tokenID); err != nil {
			err = fmt.Errorf("failed to scan row to tokenID: %w", err)
			s.base.Logger.Error(err)
			return nil, upgradeError(err)
		}
		subscriptions = append(subscriptions, big.NewInt(tokenID))
	}
	return subscriptions, nil
}

func (s *storage) IsCancellationSubscriptionExist(ctx context.Context, session eventlistener.Session, tokenID *big.Int) (bool, error) {
	tx, err := sqlite.GetSQLXTx(session)
	if err != nil {
		err = fmt.Errorf("failed to get transaction from session: %w", err)
		s.base.Logger.Error(err)
		return false, err
	}

	dbTokenID := int64(0)
	if err := tx.StmtxContext(ctx, s.getCancellationSubscription).GetContext(ctx, &dbTokenID, tokenID.Int64()); err != nil {
		if errors.Is(err, sql.ErrNoRows) { // TODO@ unify
			return false, nil
		}
		err = fmt.Errorf("failed to execute get cancellation subscription statement: %w", err)
		s.base.Logger.Error(err)
		return false, upgradeError(err)
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
		err = fmt.Errorf("failed to prepare insert cancellation subscription statement: %w", err)
		s.base.Logger.Error(err)
		return err
	}
	s.insertCancellationSubscription = insertCancellationSubscription

	removeCancellationSubscription, err := s.base.DB.PreparexContext(ctx, fmt.Sprintf(`
		DELETE FROM %s
		WHERE token_id = ?
	`, cancellationSubscriptionsTable))
	if err != nil {
		err = fmt.Errorf("failed to prepare remove cancellation subscription statement: %w", err)
		s.base.Logger.Error(err)
		return err
	}
	s.removeCancellationSubscription = removeCancellationSubscription

	getAllCancellationSubscription, err := s.base.DB.PreparexContext(ctx, fmt.Sprintf(`
		SELECT * FROM %s
	`, cancellationSubscriptionsTable))
	if err != nil {
		err = fmt.Errorf("failed to prepare get all cancellation subscriptions statement: %w", err)
		s.base.Logger.Error(err)
		return err
	}
	s.getAllCancellationSubscription = getAllCancellationSubscription

	getCancellationSubscription, err := s.base.DB.PreparexContext(ctx, fmt.Sprintf(`
		SELECT * FROM %s
		WHERE token_id = ?
	`, cancellationSubscriptionsTable))
	if err != nil {
		err = fmt.Errorf("failed to prepare get cancellation subscription statement: %w", err)
		s.base.Logger.Error(err)
		return err
	}
	s.getCancellationSubscription = getCancellationSubscription

	return nil
}
