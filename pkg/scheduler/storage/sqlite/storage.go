package sqlite

import (
	"context"
	"database/sql"
	"errors"

	"github.com/chain4travel/camino-messenger-bot/pkg/database/sqlite"
	"github.com/chain4travel/camino-messenger-bot/pkg/scheduler"
	_ "github.com/golang-migrate/migrate/v4/source/file" // required by migrate
	"github.com/jmoiron/sqlx"
	_ "github.com/mattn/go-sqlite3" // sql driver, required
	"go.uber.org/zap"
)

const dbName = "scheduler"

var (
	_ Storage                  = (*storage)(nil)
	_ scheduler.Session        = (*sqlite.SQLxTxSession)(nil)
	_ scheduler.SessionHandler = (*storage)(nil)

	ErrNotFound              = errors.New("not found")
	ErrAlreadyCommitted      = errors.New("already committed")
	ErrUnexpectedSessionType = errors.New("unexpected session type")
)

type Storage interface {
	Close() error

	scheduler.Storage
}

func New(ctx context.Context, logger *zap.SugaredLogger, cfg sqlite.DBConfig) (Storage, error) {
	baseDB, err := sqlite.New(ctx, logger, cfg, dbName)
	if err != nil {
		return nil, err
	}

	s := &storage{baseDB: baseDB}

	if err := s.prepare(ctx); err != nil {
		return nil, err
	}

	return s, nil
}

type storage struct {
	baseDB *sqlite.SQLiteXDB

	jobsStatements
}

func (s *storage) Close() error {
	return s.baseDB.Close()
}

func (s *storage) prepare(ctx context.Context) error {
	return s.prepareJobsStmts(ctx)
}

func upgradeError(err error) error {
	if errors.Is(err, sql.ErrNoRows) {
		return ErrNotFound
	}
	return err
}

func (s *storage) NewSession(ctx context.Context) (scheduler.Session, error) {
	return s.baseDB.NewSession(ctx)
}

func (s *storage) Commit(session scheduler.Session) error {
	return s.baseDB.Commit(session)
}

func (s *storage) Abort(session scheduler.Session) {
	s.baseDB.Abort(session)
}

func getSQLXTx(session scheduler.Session) (*sqlx.Tx, error) {
	s, ok := session.(sqlite.SQLxTxer)
	if !ok {
		return nil, ErrUnexpectedSessionType
	}
	return s.SQLxTx(), nil
}
