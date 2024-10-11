package sqlite

import (
	"context"
	"database/sql"
	"errors"

	"github.com/jmoiron/sqlx"
)

var (
	_ Session  = (*SQLxTxSession)(nil)
	_ SQLxTxer = (*SQLxTxSession)(nil)

	ErrAlreadyCommitted = errors.New("already committed")
)

type Session interface {
	Commit() error
	Abort() error
}

type SQLxTxer interface {
	SQLxTx() *sqlx.Tx
}

func (s *SQLiteXDB) NewSession(ctx context.Context) (*SQLxTxSession, error) {
	tx, err := s.DB.BeginTxx(ctx, &sql.TxOptions{
		Isolation: sql.LevelSerializable,
	})
	if err != nil {
		s.Logger.Error(err)
		return nil, err
	}
	return &SQLxTxSession{Tx: tx}, nil
}

func (s *SQLiteXDB) Commit(session Session) error {
	if err := session.Commit(); err != nil {
		s.Logger.Error(err)
		return err
	}
	return nil
}

func (s *SQLiteXDB) Abort(session Session) {
	if err := session.Abort(); err != nil {
		s.Logger.Error(err)
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
		return err
	}
	s.committed = true
	return nil
}

func (s *SQLxTxSession) Abort() error {
	if s.committed {
		return nil
	}
	return s.Tx.Rollback()
}

func (s *SQLxTxSession) SQLxTx() *sqlx.Tx {
	return s.Tx
}
