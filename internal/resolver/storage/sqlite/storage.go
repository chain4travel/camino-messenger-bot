// Copyright (C) 2022-2026, Chain4Travel AG. All rights reserved.
// See the file LICENSE for licensing terms.

package sqlite

import (
	"context"
	"embed"

	"github.com/chain4travel/camino-messenger-bot/v13/internal/resolver"
	"github.com/chain4travel/camino-messenger-bot/v13/pkg/database/sqlite"
	_ "github.com/golang-migrate/migrate/v4/source/file" // required by migrate
	_ "github.com/mattn/go-sqlite3"                      // sql driver, required
	"go.uber.org/zap"
)

const dbName = "event_listener"

//go:embed migrations/*.sql
var embedMigrations embed.FS

var (
	_ Storage                 = (*storage)(nil)
	_ resolver.Session        = (*sqlite.SQLxTxSession)(nil)
	_ resolver.SessionHandler = (*storage)(nil)
)

type Storage interface {
	Close() error

	resolver.Storage
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

	botsStatements
}

func (s *storage) Close() error {
	return s.base.Close()
}

func (s *storage) prepare(ctx context.Context) error {
	if err := s.prepareBotsStmts(ctx); err != nil {
		s.base.Logger.Error(err)
		return err
	}
	return nil
}

func (s *storage) NewSession(ctx context.Context) (resolver.Session, error) {
	return s.base.NewSession(ctx)
}

func (s *storage) Commit(session resolver.Session) error {
	return s.base.Commit(session)
}

func (s *storage) Abort(session resolver.Session) {
	s.base.Abort(session)
}
