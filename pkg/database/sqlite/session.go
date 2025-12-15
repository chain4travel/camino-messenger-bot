// Copyright (C) 2022-2025, Chain4Travel AG. All rights reserved.
// See the file LICENSE for licensing terms.

package sqlite

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/jmoiron/sqlx"
)

var (
	_ Session  = (*SQLxTxSession)(nil)
	_ SQLxTxer = (*SQLxTxSession)(nil)

	ErrAlreadyCommitted      = errors.New("already committed")
	ErrUnexpectedSessionType = errors.New("unexpected session type")
)

type Session interface {
	Commit() error
	Abort() error
}

type SQLxTxer interface {
	SQLxTx() *sqlx.Tx
}

func (s *DB) NewSession(ctx context.Context) (*SQLxTxSession, error) {
	tx, err := s.DB.BeginTxx(ctx, &sql.TxOptions{
		Isolation: sql.LevelSerializable,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to begin new sql transaction: %w", err)
	}
	return &SQLxTxSession{Tx: tx}, nil
}

func (s *DB) Commit(session Session) error {
	if err := session.Commit(); err != nil {
		return fmt.Errorf("failed to commit db session: %w", err)
	}
	return nil
}

func (s *DB) Abort(session Session) {
	if err := session.Abort(); err != nil {
		s.Logger.Errorf("failed to abort db session: %v", err)
	}
}

type SQLxTxSession struct {
	*sqlx.Tx
	committed bool
}

func (s *SQLxTxSession) Commit() error {
	if s.committed {
		return ErrAlreadyCommitted
	}
	if err := s.Tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit sql transaction: %w", err)
	}
	s.committed = true
	return nil
}

func (s *SQLxTxSession) Abort() error {
	if s.committed {
		return nil
	}
	if err := s.Tx.Rollback(); err != nil {
		return fmt.Errorf("failed to rollback sql transaction: %w", err)
	}
	return nil
}

func (s *SQLxTxSession) SQLxTx() *sqlx.Tx {
	return s.Tx
}

func GetSQLXTx(session any) (*sqlx.Tx, error) {
	s, ok := session.(SQLxTxer)
	if !ok {
		return nil, ErrUnexpectedSessionType
	}
	return s.SQLxTx(), nil
}
