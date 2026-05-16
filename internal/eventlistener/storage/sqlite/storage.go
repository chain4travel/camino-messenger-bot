// Copyright (C) 2022-2026, Chain4Travel AG. All rights reserved.
// See the file LICENSE for licensing terms.

package sqlite

import (
	"context"
	"embed"
	"fmt"

	"github.com/chain4travel/camino-messenger-bot/v13/internal/eventlistener"
	"github.com/chain4travel/camino-messenger-bot/v13/pkg/database/sqlite"
	_ "github.com/golang-migrate/migrate/v4/source/file" // required by migrate
	_ "github.com/mattn/go-sqlite3"                      // sql driver, required
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
		return nil, fmt.Errorf("failed to create base sqlite DB: %w", err)
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
	cancellationSubscriptionsStatements
}

func (s *storage) Close() error {
	return s.base.Close()
}

func (s *storage) prepare(ctx context.Context) error {
	if err := s.prepareTokenBoughtSubscriptionsStmts(ctx); err != nil {
		return fmt.Errorf("failed to prepare token bought subscriptions statements: %w", err)
	}
	if err := s.prepareCancellationSubscriptionsStmts(ctx); err != nil {
		return fmt.Errorf("failed to prepare cancellation subscriptions statements: %w", err)
	}
	return nil
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
