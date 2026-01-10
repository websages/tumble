package data

import (
	"fmt"
	"strings"
)

// NewStore creates a new Store based on the driver name and DSN.
// Driver can be "mysql" or "sqlite" (case-insensitive).
func NewStore(driver, dsn string) (Store, error) {
	switch strings.ToLower(driver) {
	case "mysql":
		return NewMySQLStore(dsn)
	case "sqlite", "sqlite3":
		// modernc.org/sqlite registers as "sqlite"
		// If user config says "sqlite3", we map it.
		// NOTE: NewSQLiteStore uses "sqlite" internally now.
		return NewSQLiteStore(dsn)
	default:
		return nil, fmt.Errorf("unknown database driver: %s", driver)
	}
}
