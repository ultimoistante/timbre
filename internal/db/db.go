// Package db opens the GORM database connection. The dialect is chosen at
// runtime (sqlite now, postgres later) from configuration, so application code
// stays driver-agnostic.
package db

import (
	"fmt"

	"github.com/glebarez/sqlite"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"

	"github.com/ultimoistante/timbre/internal/config"
	"github.com/ultimoistante/timbre/internal/models"
)

// Open connects to the database using the configured driver and runs migrations.
func Open(cfg *config.Config) (*gorm.DB, error) {
	var dialector gorm.Dialector

	switch cfg.DBDriver {
	case "sqlite":
		// _pragma options enable WAL + foreign keys for the pure-Go driver.
		dsn := cfg.DBDSN + "?_pragma=journal_mode(WAL)&_pragma=foreign_keys(ON)&_pragma=busy_timeout(5000)"
		dialector = sqlite.Open(dsn)
	case "postgres":
		dialector = postgres.Open(cfg.DBDSN)
	default:
		return nil, fmt.Errorf("unsupported db driver: %q", cfg.DBDriver)
	}

	gdb, err := gorm.Open(dialector, &gorm.Config{
		Logger: logger.Default.LogMode(logger.Warn),
	})
	if err != nil {
		return nil, fmt.Errorf("open db: %w", err)
	}

	if err := gdb.AutoMigrate(models.AllModels()...); err != nil {
		return nil, fmt.Errorf("automigrate: %w", err)
	}

	return gdb, nil
}
