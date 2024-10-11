package sqlite

import (
	"errors"

	"github.com/golang-migrate/migrate/v4"
	"github.com/golang-migrate/migrate/v4/database/sqlite3"
	_ "github.com/golang-migrate/migrate/v4/source/file" // required by migrate
	"github.com/jmoiron/sqlx"
	_ "github.com/mattn/go-sqlite3" // sql driver, required
	"go.uber.org/zap"
)

type DBConfig struct {
	DBPath         string
	MigrationsPath string
}

func New(logger *zap.SugaredLogger, cfg DBConfig, dbName string) (*DB, error) {
	db, err := sqlx.Open("sqlite3", cfg.DBPath)
	if err != nil {
		logger.Error(err)
		return nil, err
	}

	s := &DB{
		Logger: logger,
		DB:     db,
	}

	if err := s.migrate(dbName, cfg.MigrationsPath); err != nil {
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

func (s *DB) migrate(dbName, migrationsPath string) error {
	s.Logger.Infof("Performing db migrations...")

	driver, err := sqlite3.WithInstance(s.DB.DB, &sqlite3.Config{})
	if err != nil {
		s.Logger.Error(err)
		return err
	}

	migration, err := migrate.NewWithDatabaseInstance(migrationsPath, dbName, driver)
	if err != nil {
		s.Logger.Error(err)
		return err
	}

	version, dirty, err := migration.Version()
	if err != nil && !errors.Is(err, migrate.ErrNilVersion) {
		s.Logger.Error(err)
		return err
	}
	if dirty {
		return errors.New("database in dirty state after previous migration, requires manual fixing")
	}
	s.Logger.Infof("Migration version: %d", version)

	err = migration.Up()
	switch {
	case errors.Is(err, migrate.ErrNoChange):
		s.Logger.Infof("No migrations needed")
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
		s.Logger.Infof("New migration version: %d", newVersion)
	}

	s.Logger.Infof("Finished preforming db migrations")
	return nil
}
