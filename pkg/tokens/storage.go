// Copyright (C) 2022-2025, Chain4Travel AG. All rights reserved.
// See the file LICENSE for licensing terms.

package tokenStorage

import (
	"context"
	"database/sql"
	"errors"

	"github.com/chain4travel/camino-messenger-bot/pkg/database/sqlite"
	_ "github.com/golang-migrate/migrate/v4/source/file" // required by migrate
	"github.com/jmoiron/sqlx"
	_ "github.com/mattn/go-sqlite3" // sql driver, required
	"go.uber.org/zap"
)

const dbName = "tokens"

var (
	_ Storage = (*storage)(nil)
)

func New(ctx context.Context, logger *zap.SugaredLogger, cfg sqlite.DBConfig) (Storage, error) {
	baseDB, err := sqlite.New(logger, cfg, dbName)
	if err != nil {
		return nil, err
	}

	s := &storage{base: baseDB}

	if err := s.prepare(ctx); err != nil {
		return nil, err
	}

	return s, nil
}

func (s *storage) Close() error {
	return s.base.Close()
}

func (s *storage) prepare(ctx context.Context) error {
	// First create the table
	query := `
	CREATE TABLE IF NOT EXISTS tokens (
		token_id TEXT PRIMARY KEY,
		bought BOOLEAN NOT NULL,
		expired BOOLEAN NOT NULL,
		created_at BLOB NOT NULL,
		expires_at BLOB NOT NULL
	);`

	tx, err := s.base.DB.BeginTx(ctx, nil)
	if err != nil {
		return err
	}

	if _, err := tx.ExecContext(ctx, query); err != nil {
		tx.Rollback()
		return err
	}

	if err := tx.Commit(); err != nil {
		return err
	}

	// Then prepare the statements
	return s.prepareTokenRecordsStmts(ctx)
}

func (s *storage) NewSession(ctx context.Context) (sqlite.Session, error) {
	return s.base.NewSession(ctx)
}

func (s *storage) Commit(session sqlite.Session) error {
	return s.base.Commit(session)
}

func (s *storage) Abort(session sqlite.Session) {
	s.base.Abort(session)
}

func (s *storage) DB() *sqlx.DB {
	return s.base.DB
}

func getSQLXTx(session sqlite.Session) (*sqlx.Tx, error) {
	s, ok := session.(sqlite.SQLxTxer)
	if !ok {
		return nil, sqlite.ErrUnexpectedSessionType
	}
	return s.SQLxTx(), nil
}

func upgradeError(err error) error {
	if errors.Is(err, sql.ErrNoRows) {
		return nil
	}
	return err
}
