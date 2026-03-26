// Package gormiotdb exposes a reusable GORM dialector for Apache IoTDB.
package gormiotdb

import (
	"gorm.io/gorm"

	"github.com/wkk778/gorm-iotdb/dialector"
)

// Config configures the IoTDB dialector.
type Config = dialector.Config

// Dialector is the concrete GORM dialector implementation for IoTDB.
type Dialector = dialector.Dialector

// Migrator is the IoTDB-aware migrator used by the dialector.
type Migrator = dialector.Migrator

// TagShardFunc resolves the physical table name for a tag set.
type TagShardFunc = dialector.TagShardFunc

// Open returns a new IoTDB GORM dialector using the provided DSN.
func Open(dsn string) gorm.Dialector {
	return dialector.Open(dsn)
}

// New returns a new IoTDB GORM dialector using a structured configuration.
func New(config Config) gorm.Dialector {
	return dialector.New(config)
}
