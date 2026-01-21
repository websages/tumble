package data

import (
	"fmt"
	"strings"

	"tumble/internal/config"

	"github.com/glebarez/sqlite"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// NewStore creates a new Store based on the driver name and DSN.
// Driver can be "mysql" or "sqlite" (case-insensitive).
func NewStore(cfg *config.Config) (Store, error) {
	var dialector gorm.Dialector

	switch strings.ToLower(cfg.Driver) {
	case "mysql":
		dialector = mysql.Open(cfg.DSN())
	case "sqlite", "sqlite3":
		dialector = sqlite.Open(cfg.DSN())
	default:
		return nil, fmt.Errorf("unknown database driver: %s", cfg.Driver)
	}

	logLevel := logger.Error
	if cfg.Mode == "development" {
		logLevel = logger.Info
	}

	db, err := gorm.Open(dialector, &gorm.Config{
		Logger: logger.Default.LogMode(logLevel),
	})
	if err != nil {
		return nil, err
	}

	return NewGormStore(db), nil
}
