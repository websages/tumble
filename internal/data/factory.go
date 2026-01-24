package data

import (
	"fmt"
	"log"
	"strings"
	"time"

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

	// Use the global log writer which main.go has configured
	// regardless of whether it's stdout or a file.
	newLogger := logger.New(
		log.New(log.Writer(), "\r\n", log.LstdFlags),
		logger.Config{
			SlowThreshold:             200 * time.Millisecond,
			LogLevel:                  logLevel,
			IgnoreRecordNotFoundError: false,
			Colorful:                  false,
		},
	)

	db, err := gorm.Open(dialector, &gorm.Config{
		Logger: newLogger,
	})
	if err != nil {
		return nil, err
	}

	return NewGormStore(db), nil
}
