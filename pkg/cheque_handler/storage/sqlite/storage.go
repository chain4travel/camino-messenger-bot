package sqlite

import (
	"context"
	"database/sql"
	"errors"

	chequeHandler "github.com/chain4travel/camino-messenger-bot/pkg/cheque_handler"
	"github.com/chain4travel/camino-messenger-bot/pkg/database"
	"github.com/chain4travel/camino-messenger-bot/pkg/database/sqlite"
	_ "github.com/golang-migrate/migrate/v4/source/file" // required by migrate
	"github.com/jmoiron/sqlx"
	_ "github.com/mattn/go-sqlite3" // sql driver, required
	"go.uber.org/zap"
)

const dbName = "cheque_handler"

var (
	_ Storage                      = (*storage)(nil)
	_ chequeHandler.Session        = (*sqlite.SQLxTxSession)(nil)
	_ chequeHandler.SessionHandler = (*storage)(nil)
)

type Storage interface {
	Close() error

	chequeHandler.Storage
}

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

type storage struct {
	base *sqlite.DB

	issuedChequeRecordsStatements
	chequeRecordsStatements
}

func (s *storage) Close() error {
	return s.base.Close()
}

func (s *storage) prepare(ctx context.Context) error {
	return errors.Join(
		s.prepareIssuedChequeRecordsStmts(ctx),
		s.prepareChequeRecordsStmts(ctx),
	)
}

func (s *storage) NewSession(ctx context.Context) (chequeHandler.Session, error) {
	return s.base.NewSession(ctx)
}

func (s *storage) Commit(session chequeHandler.Session) error {
	return s.base.Commit(session)
}

func (s *storage) Abort(session chequeHandler.Session) {
	s.base.Abort(session)
}

func getSQLXTx(session chequeHandler.Session) (*sqlx.Tx, error) {
	s, ok := session.(sqlite.SQLxTxer)
	if !ok {
		return nil, sqlite.ErrUnexpectedSessionType
	}
	return s.SQLxTx(), nil
}

func upgradeError(err error) error {
	if errors.Is(err, sql.ErrNoRows) {
		return database.ErrNotFound
	}
	return err
}
