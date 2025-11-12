// Copyright (C) 2022-2025, Chain4Travel AG. All rights reserved.
// See the file LICENSE for licensing terms.

package sqlite

import (
	"context"
	"embed"
	"errors"

	"github.com/chain4travel/camino-messenger-bot/v12/internal/messaging/encoding"
	"github.com/chain4travel/camino-messenger-bot/v12/pkg/database/sqlite"
	_ "github.com/golang-migrate/migrate/v4/source/file" // required by migrate
	_ "github.com/mattn/go-sqlite3"                      // sql driver, required
	"go.uber.org/zap"
)

const dbName = "encoder_decoder"

//go:embed migrations/*.sql
var embedMigrations embed.FS

var (
	_ Storage                 = (*storage)(nil)
	_ encoding.Session        = (*sqlite.SQLxTxSession)(nil)
	_ encoding.SessionHandler = (*storage)(nil)
)

type Storage interface {
	Close() error

	encoding.Storage
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

	pubKeysStatements
}

func (s *storage) Close() error {
	return s.base.Close()
}

func (s *storage) prepare(ctx context.Context) error {
	return errors.Join(
		s.preparePubKeysStmts(ctx),
	)
}

func (s *storage) NewSession(ctx context.Context) (encoding.Session, error) {
	return s.base.NewSession(ctx)
}

func (s *storage) Commit(session encoding.Session) error {
	return s.base.Commit(session)
}

func (s *storage) Abort(session encoding.Session) {
	s.base.Abort(session)
}
