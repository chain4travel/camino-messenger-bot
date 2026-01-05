// Copyright (C) 2022-2026, Chain4Travel AG. All rights reserved.
// See the file LICENSE for licensing terms.

package sqlite

import (
	"context"
	"database/sql"
	"embed"
	"errors"

	"github.com/chain4travel/camino-messenger-bot/v12/pkg/database/sqlite"
	"github.com/chain4travel/camino-messenger-bot/v12/pkg/scheduler"
	_ "github.com/golang-migrate/migrate/v4/source/file" // required by migrate
	_ "github.com/mattn/go-sqlite3"                      // sql driver, required
	"go.uber.org/zap"
)

const dbName = "scheduler"

//go:embed migrations/*.sql
var embedMigrations embed.FS

var (
	_ Storage                  = (*storage)(nil)
	_ scheduler.Session        = (*sqlite.SQLxTxSession)(nil)
	_ scheduler.SessionHandler = (*storage)(nil)
)

type Storage interface {
	Close() error

	scheduler.Storage
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

	jobsStatements
}

func (s *storage) Close() error {
	return s.base.Close()
}

func (s *storage) prepare(ctx context.Context) error {
	return s.prepareJobsStmts(ctx)
}

func (s *storage) NewSession(ctx context.Context) (scheduler.Session, error) {
	return s.base.NewSession(ctx)
}

func (s *storage) Commit(session scheduler.Session) error {
	return s.base.Commit(session)
}

func (s *storage) Abort(session scheduler.Session) {
	s.base.Abort(session)
}

func upgradeError(err error) error {
	if errors.Is(err, sql.ErrNoRows) {
		return scheduler.ErrNotFound
	}
	return err
}
