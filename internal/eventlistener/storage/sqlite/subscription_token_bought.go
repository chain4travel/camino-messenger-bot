// Copyright (C) 2022-2026, Chain4Travel AG. All rights reserved.
// See the file LICENSE for licensing terms.

package sqlite

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"math/big"
	"time"

	"github.com/chain4travel/camino-messenger-bot/v13/internal/eventlistener"
	"github.com/chain4travel/camino-messenger-bot/v13/pkg/database/sqlite"
	"github.com/jmoiron/sqlx"
)

const tokenBoughtSubscriptionsTable = "token_bought_subscriptions"

var _ eventlistener.Storage = (*storage)(nil)

type tokenBoughtSubscription struct {
	TokenID int64  `db:"token_id"`
	MintID  string `db:"mint_id"`
	Timeout int64  `db:"timeout"`
}

func (s *storage) AddTokenBoughtSubscription(ctx context.Context, session eventlistener.Session, subscription *eventlistener.TokenBoughtSubscription) error {
	tx, err := sqlite.GetSQLXTx(session)
	if err != nil {
		return fmt.Errorf("failed to get transaction from db session: %w", err)
	}

	result, err := tx.NamedStmtContext(ctx, s.insertTokenBoughtSubscription).
		ExecContext(ctx, tokenBoughtSubscriptionFromModel(subscription))
	if err != nil {
		return fmt.Errorf("failed to execute insert token bought subscription statement: %w", err)
	}
	if rowsAffected, err := result.RowsAffected(); err != nil {
		return fmt.Errorf("failed to get rowsAffected from statement execution result: %w", err)
	} else if rowsAffected != 1 {
		return fmt.Errorf("unexpected number of rows affected: expected 1, but affected %d", rowsAffected)
	}
	return nil
}

func (s *storage) RemoveTokenBoughtSubscription(ctx context.Context, session eventlistener.Session, tokenID *big.Int) error {
	tx, err := sqlite.GetSQLXTx(session)
	if err != nil {
		return fmt.Errorf("failed to get transaction from db session: %w", err)
	}

	result, err := tx.StmtxContext(ctx, s.removeTokenBoughtSubscription).ExecContext(ctx, tokenID.Uint64())
	if err != nil {
		return fmt.Errorf("failed to execute remove token bought subscription statement: %w", err)
	}
	if rowsAffected, err := result.RowsAffected(); err != nil {
		return fmt.Errorf("failed to get rowsAffected from statement execution result: %w", err)
	} else if rowsAffected != 1 {
		return fmt.Errorf("unexpected number of rows affected: expected 1, but affected %d", rowsAffected)
	}
	return nil
}

func (s *storage) GetAllTokenBoughtSubscriptions(ctx context.Context, session eventlistener.Session) ([]eventlistener.TokenBoughtSubscription, error) {
	tx, err := sqlite.GetSQLXTx(session)
	if err != nil {
		return nil, fmt.Errorf("failed to get transaction from db session: %w", err)
	}

	subscriptions := []eventlistener.TokenBoughtSubscription{}
	rows, err := tx.StmtxContext(ctx, s.getAllTokenBoughtSubscription).QueryxContext(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to execute get all token bought subscriptions statement: %w", err)
	}
	for rows.Next() {
		subscription := &tokenBoughtSubscription{}
		if err := rows.StructScan(subscription); err != nil {
			return nil, fmt.Errorf("failed to scan row to tokenBoughtSubscription: %w", err)
		}
		subscriptions = append(subscriptions, *modelFromTokenBoughtSubscription(subscription))
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error occurred during rows iteration: %w", err)
	}
	return subscriptions, nil
}

func (s *storage) GetTokenBoughtSubscription(ctx context.Context, session eventlistener.Session, tokenID *big.Int) (*eventlistener.TokenBoughtSubscription, error) {
	tx, err := sqlite.GetSQLXTx(session)
	if err != nil {
		return nil, fmt.Errorf("failed to get transaction from db session: %w", err)
	}

	subscription := &tokenBoughtSubscription{}
	if err := tx.StmtxContext(ctx, s.getTokenBoughtSubscription).GetContext(ctx, subscription, tokenID.Int64()); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, eventlistener.ErrNotFound
		}
		return nil, fmt.Errorf("failed to execute get token bought subscription statement: %w", err)
	}
	return modelFromTokenBoughtSubscription(subscription), nil
}

func (s *storage) GetTokenBoughtSubscriptionByMinTimeout(ctx context.Context, session eventlistener.Session) (*eventlistener.TokenBoughtSubscription, error) {
	tx, err := sqlite.GetSQLXTx(session)
	if err != nil {
		return nil, fmt.Errorf("failed to get transaction from db session: %w", err)
	}
	subscription := &tokenBoughtSubscription{}
	if err := tx.StmtxContext(ctx, s.getTokenBoughtSubscriptionByMinTimeout).GetContext(ctx, subscription); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, eventlistener.ErrNotFound
		}
		return nil, fmt.Errorf("failed to execute get token bought subscription by min timeout statement: %w", err)
	}
	return modelFromTokenBoughtSubscription(subscription), nil
}

type tokenBoughtSubscriptionsStatements struct {
	insertTokenBoughtSubscription          *sqlx.NamedStmt
	removeTokenBoughtSubscription          *sqlx.Stmt
	getAllTokenBoughtSubscription          *sqlx.Stmt
	getTokenBoughtSubscription             *sqlx.Stmt
	getTokenBoughtSubscriptionByMinTimeout *sqlx.Stmt
}

func (s *storage) prepareTokenBoughtSubscriptionsStmts(ctx context.Context) error {
	insertTokenBoughtSubscription, err := s.base.DB.PrepareNamedContext(ctx, fmt.Sprintf(`
		INSERT INTO %s (
			token_id,
			mint_id,
			timeout
		) VALUES (
			:token_id,
			:mint_id,
			:timeout
		)
	`, tokenBoughtSubscriptionsTable))
	if err != nil {
		return fmt.Errorf("failed to prepare insert token bought subscription statement: %w", err)
	}
	s.insertTokenBoughtSubscription = insertTokenBoughtSubscription

	removeTokenBoughtSubscription, err := s.base.DB.PreparexContext(ctx, fmt.Sprintf(`
		DELETE FROM %s
		WHERE token_id = ?
	`, tokenBoughtSubscriptionsTable))
	if err != nil {
		return fmt.Errorf("failed to prepare remove token bought subscription statement: %w", err)
	}
	s.removeTokenBoughtSubscription = removeTokenBoughtSubscription

	getAllTokenBoughtSubscription, err := s.base.DB.PreparexContext(ctx, fmt.Sprintf(`
		SELECT * FROM %s
	`, tokenBoughtSubscriptionsTable))
	if err != nil {
		return fmt.Errorf("failed to prepare get all token bought subscriptions statement: %w", err)
	}
	s.getAllTokenBoughtSubscription = getAllTokenBoughtSubscription

	getTokenBoughtSubscription, err := s.base.DB.PreparexContext(ctx, fmt.Sprintf(`
		SELECT * FROM %s
		WHERE token_id = ?
	`, tokenBoughtSubscriptionsTable))
	if err != nil {
		return fmt.Errorf("failed to prepare get token bought subscription statement: %w", err)
	}
	s.getTokenBoughtSubscription = getTokenBoughtSubscription

	getTokenBoughtSubscriptionByMinTimeout, err := s.base.DB.PreparexContext(ctx, fmt.Sprintf(`
		SELECT * FROM %s
		WHERE timeout = (
			SELECT MIN(timeout) FROM %s
		)
	`, tokenBoughtSubscriptionsTable, tokenBoughtSubscriptionsTable))
	if err != nil {
		return fmt.Errorf("failed to prepare get token bought subscription by min timeout statement: %w", err)
	}
	s.getTokenBoughtSubscriptionByMinTimeout = getTokenBoughtSubscriptionByMinTimeout

	return nil
}

func modelFromTokenBoughtSubscription(subscription *tokenBoughtSubscription) *eventlistener.TokenBoughtSubscription {
	return &eventlistener.TokenBoughtSubscription{
		TokenID: big.NewInt(subscription.TokenID),
		MintID:  subscription.MintID,
		Timeout: time.Unix(subscription.Timeout, 0),
	}
}

func tokenBoughtSubscriptionFromModel(model *eventlistener.TokenBoughtSubscription) *tokenBoughtSubscription {
	return &tokenBoughtSubscription{
		TokenID: model.TokenID.Int64(),
		MintID:  model.MintID,
		Timeout: model.Timeout.Unix(),
	}
}
