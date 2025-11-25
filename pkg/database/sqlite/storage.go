// Copyright (C) 2022-2025, Chain4Travel AG. All rights reserved.
// See the file LICENSE for licensing terms.

package sqlite

import (
	"errors"
	"io/fs"
	"os"
	"path/filepath"

	"github.com/golang-migrate/migrate/v4"
	"github.com/golang-migrate/migrate/v4/database/sqlite3"
	_ "github.com/golang-migrate/migrate/v4/source/file" // required by migrate
	"github.com/golang-migrate/migrate/v4/source/iofs"
	"github.com/jmoiron/sqlx"
	_ "github.com/mattn/go-sqlite3" // sql driver, required
	"go.uber.org/zap"
)

type DBConfig struct {
	DBPath string
}

func New(logger *zap.SugaredLogger, migrationsFS fs.FS, dbPath string, dbName string) (*DB, error) {
	if err := os.MkdirAll(filepath.Dir(dbPath), os.ModePerm); err != nil {
		logger.Error(err)
		return nil, err
	}

	db, err := sqlx.Open("sqlite3", dbPath+".sqlite")
	if err != nil {
		logger.Error(err)
		return nil, err
	}

	s := &DB{
		Logger: logger,
		DB:     db,
	}

	if err := s.migrate(migrationsFS, dbName, false); err != nil {
		return nil, err
	}

	return s, nil
}

type DB struct {
	Logger *zap.SugaredLogger
	DB     *sqlx.DB
}

func (s *DB) Close() error {
	if err := s.DB.Close(); err != nil {
		s.Logger.Error(err)
		return err
	}
	return nil
}

var _ migrate.Logger = (*migrationLogger)(nil)

type migrationLogger struct {
	*zap.SugaredLogger
}

func (l *migrationLogger) Printf(format string, v ...interface{}) {
	l.Infof(format, v...)
}

func (l *migrationLogger) Verbose() bool {
	return false
}

func (s *DB) migrate(migrationsFS fs.FS, dbName string, logMigrations bool) error {
	s.Logger.Infof("Performing db migrations for %s...", dbName)

	driver, err := sqlite3.WithInstance(s.DB.DB, &sqlite3.Config{})
	if err != nil {
		s.Logger.Error(err)
		return err
	}

	src, err := iofs.New(migrationsFS, "migrations")
	if err != nil {
		s.Logger.Error(err)
		return err
	}

	migration, err := migrate.NewWithInstance("iofs", src, dbName, driver)
	if err != nil {
		s.Logger.Error(err)
		return err
	}

	if logMigrations {
		migration.Log = &migrationLogger{s.Logger}
	}

	version, dirty, err := migration.Version()
	if err != nil && !errors.Is(err, migrate.ErrNilVersion) {
		s.Logger.Error(err)
		return err
	}
	if dirty {
		return errors.New("database in dirty state after previous migration, requires manual fixing")
	}
	s.Logger.Infof("%s db migration version: %d", dbName, version)

	err = migration.Up()
	switch {
	case errors.Is(err, migrate.ErrNoChange):
		s.Logger.Infof("No migrations needed for %s database", dbName)
	case err != nil:
		s.Logger.Error(err)
		return err
	default:
		newVersion, dirty, err := migration.Version()
		if err != nil && !errors.Is(err, migrate.ErrNilVersion) {
			s.Logger.Error(err)
			return err
		}
		if dirty {
			return errors.New("database in dirty state after previous migration, requires manual fixing")
		}
		s.Logger.Infof("New %s db migration version: %d", dbName, newVersion)
	}

	s.Logger.Infof("Finished performing db migrations for %s", dbName)
	return nil
}
