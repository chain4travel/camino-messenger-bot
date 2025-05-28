// Copyright (C) 2022-2025, Chain4Travel AG. All rights reserved.
// See the file LICENSE for licensing terms.

package sqlite

import (
	"context"
	"database/sql"
	"embed"
	"errors"

	eventlistener "github.com/chain4travel/camino-messenger-bot/v11/internal/event_listener"
	"github.com/chain4travel/camino-messenger-bot/v11/pkg/database/sqlite"
	_ "github.com/golang-migrate/migrate/v4/source/file" // required by migrate
	"github.com/jmoiron/sqlx"
	_ "github.com/mattn/go-sqlite3" // sql driver, required
	"go.uber.org/zap"
)

const dbName = "event_listener"

//go:embed migrations/*.sql
var embedMigrations embed.FS

var (
	_ Storage                      = (*storage)(nil)
	_ eventlistener.Session        = (*sqlite.SQLxTxSession)(nil)
	_ eventlistener.SessionHandler = (*storage)(nil)
)

type Storage interface {
	Close() error

	eventlistener.Storage
}

func New(ctx context.Context, logger *zap.SugaredLogger, dbPath string) (Storage, error) {
	baseDB, err := sqlite.New(logger, embedMigrations, dbPath, dbName)
	if err != nil {
		return nil, err
	}

	s := &storage{base: baseDB}

	if err := s.prepare(ctx); err != nil {
		return nil, err
	}

	return s, nil
}

type storage struct {
	base *sqlite.DB

	tokenBoughtSubscriptionsStatements
}

func (s *storage) Close() error {
	return s.base.Close()
}

func (s *storage) prepare(ctx context.Context) error {
	return s.prepareTokenBoughtSubscriptionsStmts(ctx)
}

func (s *storage) NewSession(ctx context.Context) (eventlistener.Session, error) {
	return s.base.NewSession(ctx)
}

func (s *storage) Commit(session eventlistener.Session) error {
	return s.base.Commit(session)
}

func (s *storage) Abort(session eventlistener.Session) {
	s.base.Abort(session)
}

func getSQLXTx(session eventlistener.Session) (*sqlx.Tx, error) {
	s, ok := session.(sqlite.SQLxTxer)
	if !ok {
		return nil, sqlite.ErrUnexpectedSessionType
	}
	return s.SQLxTx(), nil
}

func upgradeError(err error) error {
	if errors.Is(err, sql.ErrNoRows) {
		return eventlistener.ErrNotFound
	}
	return err
}
