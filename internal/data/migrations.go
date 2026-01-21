package data

import (
	"database/sql"
	"fmt"
	"log/slog"

	"github.com/golang-migrate/migrate/v4"
	"github.com/golang-migrate/migrate/v4/database"
	"github.com/golang-migrate/migrate/v4/database/mysql"
	"github.com/golang-migrate/migrate/v4/source/iofs"

	tumblesql "tumble/sql"
)

// RunMigrations executes pending migrations.
// driverName should be "mysql" or "sqlite"
func RunMigrations(db *sql.DB, driverName string, databaseName string) error {
	slog.Info("Running migrations", "driver", driverName)

	sourceDriver, err := iofs.New(tumblesql.MigrationFS, driverName)
	if err != nil {
		return fmt.Errorf("failed to create iofs source: %w", err)
	}

	var databaseDriver database.Driver

	switch driverName {
	case "mysql":
		databaseDriver, err = mysql.WithInstance(db, &mysql.Config{})
		if err != nil {
			return fmt.Errorf("failed to create mysql driver: %w", err)
		}
	case "sqlite":
		// Custom driver for modernc/sqlite (no CGO)
		// We implement this in sqlite_driver.go
		databaseDriver, err = WithInstance(db, &Config{})
		if err != nil {
			return fmt.Errorf("failed to create sqlite driver: %w", err)
		}
	default:
		return fmt.Errorf("unsupported driver: %s", driverName)
	}

	m, err := migrate.NewWithInstance(
		"iofs", sourceDriver,
		driverName, databaseDriver,
	)
	if err != nil {
		return fmt.Errorf("failed to create migrate instance: %w", err)
	}

	if err := m.Up(); err != nil && err != migrate.ErrNoChange {
		return fmt.Errorf("failed to run up migrations: %w", err)
	}

	slog.Info("Migrations completed successfully")
	return nil
}
