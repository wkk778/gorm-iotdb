// Package driver provides a thin database/sql wrapper for the upstream IoTDB driver.
package driver

import (
	"database/sql"
	"time"

	_ "github.com/wkk778/gorm-iotdb/driver/iotdbsql"
)

// DriverName is the registered database/sql driver name exposed by the upstream client.
const DriverName = "iotdb"

// Config configures the shared database/sql pool used by the dialector.
type Config struct {
	DSN             string
	DriverName      string
	MaxOpenConns    int
	MaxIdleConns    int
	ConnMaxIdleTime time.Duration
	ConnMaxLifetime time.Duration
}

// Open creates an IoTDB-backed database/sql pool from the provided configuration.
func Open(config Config) (*sql.DB, error) {
	name := config.DriverName
	if name == "" {
		name = DriverName
	}

	db, err := sql.Open(name, config.DSN)
	if err != nil {
		return nil, err
	}

	if config.MaxOpenConns > 0 {
		db.SetMaxOpenConns(config.MaxOpenConns)
	}
	if config.MaxIdleConns > 0 {
		db.SetMaxIdleConns(config.MaxIdleConns)
	}
	if config.ConnMaxIdleTime > 0 {
		db.SetConnMaxIdleTime(config.ConnMaxIdleTime)
	}
	if config.ConnMaxLifetime > 0 {
		db.SetConnMaxLifetime(config.ConnMaxLifetime)
	}

	return db, nil
}
