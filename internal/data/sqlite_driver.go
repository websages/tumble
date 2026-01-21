package data

import (
	"database/sql"
	"fmt"
	"io"
	"io/ioutil"
	"strings"

	migrate "github.com/golang-migrate/migrate/v4/database"
)

func init() {
	// We don't register automatically to avoid importing this package everywhere
}

type Config struct {
	MigrationsTable string
	NoTxWrap        bool
}

type SQLite struct {
	db     *sql.DB
	config *Config
}

func WithInstance(instance *sql.DB, config *Config) (migrate.Driver, error) {
	if config == nil {
		config = &Config{}
	}
	if config.MigrationsTable == "" {
		config.MigrationsTable = "schema_migrations"
	}
	return &SQLite{
		db:     instance,
		config: config,
	}, nil
}

func (s *SQLite) Open(url string) (migrate.Driver, error) {
	return nil, fmt.Errorf("not implemented, use WithInstance")
}

func (s *SQLite) Close() error {
	// We don't close the DB here as it's passed in instance
	return nil
}

func (s *SQLite) Lock() error {
	return nil // SQLite doesn't need explicit locking usually if single threaded migration or handled by db lock
}

func (s *SQLite) Unlock() error {
	return nil
}

func (s *SQLite) Run(migration io.Reader) error {
	migr, err := ioutil.ReadAll(migration)
	if err != nil {
		return err
	}
	query := string(migr)
	if strings.TrimSpace(query) == "" {
		return nil
	}

	if s.config.NoTxWrap {
		_, err := s.db.Exec(query)
		return err
	}

	tx, err := s.db.Begin()
	if err != nil {
		return err
	}
	if _, err := tx.Exec(query); err != nil {
		tx.Rollback()
		return err
	}
	return tx.Commit()
}

func (s *SQLite) SetVersion(version int, dirty bool) error {
	tx, err := s.db.Begin()
	if err != nil {
		return err
	}
	// Delete all
	if _, err := tx.Exec(fmt.Sprintf("DELETE FROM %s", s.config.MigrationsTable)); err != nil {
		tx.Rollback()
		return err
	}
	// Insert new
	if _, err := tx.Exec(fmt.Sprintf("INSERT INTO %s (version, dirty) VALUES (?, ?)", s.config.MigrationsTable), version, dirty); err != nil {
		tx.Rollback()
		return err
	}
	return tx.Commit()
}

func (s *SQLite) Version() (version int, dirty bool, err error) {
	query := fmt.Sprintf("SELECT version, dirty FROM %s LIMIT 1", s.config.MigrationsTable)
	err = s.db.QueryRow(query).Scan(&version, &dirty)
	if err != nil {
		if err == sql.ErrNoRows {
			return migrate.NilVersion, false, nil
		}
		// If table doesn't exist, create it and return nil version
		// simple check: try creating
		if err := s.ensureVersionTable(); err != nil {
			return 0, false, err
		}
		// Retry
		err = s.db.QueryRow(query).Scan(&version, &dirty)
		if err == sql.ErrNoRows {
			return migrate.NilVersion, false, nil
		}
		return version, dirty, err
	}
	return version, dirty, nil
}

func (s *SQLite) Drop() error {
	// Not implemented
	return nil
}

func (s *SQLite) ensureVersionTable() error {
	query := fmt.Sprintf(`
		CREATE TABLE IF NOT EXISTS %s (
			version INTEGER PRIMARY KEY,
			dirty BOOLEAN NOT NULL
		)
	`, s.config.MigrationsTable)
	_, err := s.db.Exec(query)
	return err
}
