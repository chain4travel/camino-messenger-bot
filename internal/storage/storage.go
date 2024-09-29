package storage

import (
	"context"
	"database/sql"
	"errors"

	"github.com/golang-migrate/migrate/v4"
	"github.com/golang-migrate/migrate/v4/database/sqlite3"
	_ "github.com/golang-migrate/migrate/v4/source/file" // required by migrate
	"github.com/jmoiron/sqlx"
	_ "github.com/mattn/go-sqlite3" // sql driver, required
	"go.uber.org/zap"
)

var (
	_ Storage = (*storage)(nil)
	_ Session = (*session)(nil)

	ErrNotFound         = errors.New("not found")
	ErrAlreadyCommitted = errors.New("already committed")
)

type Storage interface {
	NewSession(ctx context.Context) (Session, error)
	Close(ctx context.Context) error
}

type Session interface {
	Commit() error
	Abort()

	JobsStorage
	ChequeRecordsStorage
	IssuedChequeRecordsStorage
}

func New(ctx context.Context, logger *zap.SugaredLogger, dbPath, dbName, migrationsPath string) (Storage, error) {
	db, err := sqlx.Open("sqlite3", dbPath)
	if err != nil {
		logger.Error(err)
		return nil, err
	}

	s := &storage{
		logger: logger,
		db:     db,
	}

	if err := s.migrate(ctx, dbName, migrationsPath); err != nil {
		return nil, err
	}

	if err := s.prepare(ctx); err != nil {
		return nil, err
	}

	return s, nil
}

type storage struct {
	logger *zap.SugaredLogger
	db     *sqlx.DB

	issuedChequeRecordsStatements
	chequeRecordsStatements
	jobsStatements
}

func (s *storage) migrate(_ context.Context, dbName, migrationsPath string) error {
	s.logger.Infof("Performing db migrations...")

	driver, err := sqlite3.WithInstance(s.db.DB, &sqlite3.Config{})
	if err != nil {
		s.logger.Error(err)
		return err
	}

	migration, err := migrate.NewWithDatabaseInstance(migrationsPath, dbName, driver)
	if err != nil {
		s.logger.Error(err)
		return err
	}

	version, dirty, err := migration.Version()
	if err != nil && !errors.Is(err, migrate.ErrNilVersion) {
		s.logger.Error(err)
		return err
	}
	if dirty {
		return errors.New("database in dirty state after previous migration, requires manual fixing")
	}
	s.logger.Infof("Migration version: %d", version)

	err = migration.Up()
	switch {
	case errors.Is(err, migrate.ErrNoChange):
		s.logger.Infof("No migrations needed")
	case err != nil:
		s.logger.Error(err)
		return err
	default:
		newVersion, dirty, err := migration.Version()
		if err != nil && !errors.Is(err, migrate.ErrNilVersion) {
			s.logger.Error(err)
			return err
		}
		if dirty {
			return errors.New("database in dirty state after previous migration, requires manual fixing")
		}
		s.logger.Infof("New migration version: %d", newVersion)
	}

	s.logger.Infof("Finished preforming db migrations")
	return nil
}

func (s *storage) prepare(ctx context.Context) error {
	return errors.Join(
		s.prepareIssuedChequeRecordsStmts(ctx),
		s.prepareChequeRecordsStmts(ctx),
		s.prepareJobsStmts(ctx),
	)
}

func (s *storage) Close(_ context.Context) error {
	if err := s.db.Close(); err != nil {
		s.logger.Error(err)
		return err
	}
	return nil
}

func (s *storage) NewSession(ctx context.Context) (Session, error) {
	tx, err := s.db.BeginTxx(ctx, &sql.TxOptions{
		Isolation: sql.LevelSerializable,
	})
	if err != nil {
		s.logger.Error(err)
		return nil, err
	}
	return &session{tx: tx, logger: s.logger, storage: s}, nil
}

type session struct {
	storage   *storage
	logger    *zap.SugaredLogger
	tx        *sqlx.Tx
	committed bool
}

func (s *session) Commit() error {
	if s.committed {
		return ErrAlreadyCommitted
	}
	if err := s.tx.Commit(); err != nil {
		s.logger.Error(err)
		return upgradeError(err)
	}
	s.committed = true
	return nil
}

func (s *session) Abort() {
	if s.committed {
		return
	}
	if err := s.tx.Rollback(); err != nil {
		s.logger.Error(err)
	}
}

func upgradeError(err error) error {
	if errors.Is(err, sql.ErrNoRows) {
		return ErrNotFound
	}
	return err
}

func upgradeErrorAllowNotFound(err error) error { //nolint:unused
	if errors.Is(err, sql.ErrNoRows) {
		return nil
	}
	return err
}
