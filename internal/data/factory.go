package data

import (
	"fmt"
	"strings"

	"github.com/glebarez/sqlite"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// NewStore creates a new Store based on the driver name and DSN.
// Driver can be "mysql" or "sqlite" (case-insensitive).
func NewStore(driver, dsn string) (Store, error) {
	var dialector gorm.Dialector

	switch strings.ToLower(driver) {
	case "mysql":
		dialector = mysql.Open(dsn)
	case "sqlite", "sqlite3":
		dialector = sqlite.Open(dsn)
	default:
		return nil, fmt.Errorf("unknown database driver: %s", driver)
	}

	db, err := gorm.Open(dialector, &gorm.Config{
		Logger: logger.Default.LogMode(logger.Info),
	})
	if err != nil {
		return nil, err
	}

	return NewGormStore(db), nil
}
