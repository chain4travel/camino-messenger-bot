// Copyright (C) 2022-2025, Chain4Travel AG. All rights reserved.
// See the file LICENSE for licensing terms.

package sqlite

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"math/big"
	"time"

	"github.com/chain4travel/camino-messenger-bot/v11/internal/eventlistener"
	"github.com/chain4travel/camino-messenger-bot/v11/pkg/database/sqlite"
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
		s.base.Logger.Error(err)
		return err
	}

	result, err := tx.NamedStmtContext(ctx, s.insertTokenBoughtSubscription).
		ExecContext(ctx, tokenBoughtSubscriptionFromModel(subscription))
	if err != nil {
		s.base.Logger.Error(err)
		return upgradeError(err)
	}
	if rowsAffected, err := result.RowsAffected(); err != nil {
		s.base.Logger.Error(err)
		return upgradeError(err)
	} else if rowsAffected != 1 {
		return fmt.Errorf("failed to add token bought subscription: expected to affect 1 row, but affected %d", rowsAffected)
	}
	return nil
}

func (s *storage) RemoveTokenBoughtSubscription(ctx context.Context, session eventlistener.Session, tokenID *big.Int) error {
	tx, err := sqlite.GetSQLXTx(session)
	if err != nil {
		s.base.Logger.Error(err)
		return err
	}

	result, err := tx.StmtxContext(ctx, s.removeTokenBoughtSubscription).ExecContext(ctx, tokenID.Uint64())
	if err != nil {
		s.base.Logger.Error(err)
		return upgradeError(err)
	}
	if rowsAffected, err := result.RowsAffected(); err != nil {
		s.base.Logger.Error(err)
		return upgradeError(err)
	} else if rowsAffected != 1 {
		return fmt.Errorf("error removing token bought subscription: expected to affect 1 row, but affected %d", rowsAffected)
	}
	return nil
}

func (s *storage) GetAllTokenBoughtSubscriptions(ctx context.Context, session eventlistener.Session) ([]eventlistener.TokenBoughtSubscription, error) {
	tx, err := sqlite.GetSQLXTx(session)
	if err != nil {
		s.base.Logger.Error(err)
		return nil, err
	}

	subscriptions := []eventlistener.TokenBoughtSubscription{}
	rows, err := tx.StmtxContext(ctx, s.getAllTokenBoughtSubscription).QueryxContext(ctx)
	if err != nil {
		s.base.Logger.Error(err)
		return nil, upgradeError(err)
	}
	for rows.Next() {
		subscription := &tokenBoughtSubscription{}
		if err := rows.StructScan(subscription); err != nil {
			s.base.Logger.Errorf("failed to get token bought subscription from db: %v", err)
			return nil, upgradeError(err)
		}
		subscriptions = append(subscriptions, *modelFromTokenBoughtSubscription(subscription))
	}
	return subscriptions, nil
}

func (s *storage) GetTokenBoughtSubscription(ctx context.Context, session eventlistener.Session, tokenID *big.Int) (*eventlistener.TokenBoughtSubscription, error) {
	tx, err := sqlite.GetSQLXTx(session)
	if err != nil {
		s.base.Logger.Error(err)
		return nil, err
	}

	subscription := &tokenBoughtSubscription{}
	if err := tx.StmtxContext(ctx, s.getTokenBoughtSubscription).GetContext(ctx, subscription, tokenID.Int64()); err != nil {
		if !errors.Is(err, sql.ErrNoRows) {
			s.base.Logger.Error(err)
		}
		return nil, upgradeError(err)
	}
	return modelFromTokenBoughtSubscription(subscription), nil
}

func (s *storage) GetTokenBoughtSubscriptionByMinTimeout(ctx context.Context, session eventlistener.Session) (*eventlistener.TokenBoughtSubscription, error) {
	tx, err := sqlite.GetSQLXTx(session)
	if err != nil {
		s.base.Logger.Error(err)
		return nil, err
	}
	subscription := &tokenBoughtSubscription{}
	if err := tx.StmtxContext(ctx, s.getTokenBoughtSubscriptionByMinTimeout).GetContext(ctx, subscription); err != nil {
		if !errors.Is(err, sql.ErrNoRows) {
			s.base.Logger.Error(err)
		}
		return nil, upgradeError(err)
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
		s.base.Logger.Error(err)
		return err
	}
	s.insertTokenBoughtSubscription = insertTokenBoughtSubscription

	removeTokenBoughtSubscription, err := s.base.DB.PreparexContext(ctx, fmt.Sprintf(`
		DELETE FROM %s
		WHERE token_id = ?
	`, tokenBoughtSubscriptionsTable))
	if err != nil {
		s.base.Logger.Error(err)
		return err
	}
	s.removeTokenBoughtSubscription = removeTokenBoughtSubscription

	getAllTokenBoughtSubscription, err := s.base.DB.PreparexContext(ctx, fmt.Sprintf(`
		SELECT * FROM %s
	`, tokenBoughtSubscriptionsTable))
	if err != nil {
		s.base.Logger.Error(err)
		return err
	}
	s.getAllTokenBoughtSubscription = getAllTokenBoughtSubscription

	getTokenBoughtSubscription, err := s.base.DB.PreparexContext(ctx, fmt.Sprintf(`
		SELECT * FROM %s
		WHERE token_id = ?
	`, tokenBoughtSubscriptionsTable))
	if err != nil {
		s.base.Logger.Error(err)
		return err
	}
	s.getTokenBoughtSubscription = getTokenBoughtSubscription

	getTokenBoughtSubscriptionByMinTimeout, err := s.base.DB.PreparexContext(ctx, fmt.Sprintf(`
		SELECT * FROM %s
		WHERE timeout = (
			SELECT MIN(timeout) FROM %s
		)
	`, tokenBoughtSubscriptionsTable, tokenBoughtSubscriptionsTable))
	if err != nil {
		s.base.Logger.Error(err)
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
