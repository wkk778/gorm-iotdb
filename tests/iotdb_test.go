package tests

import (
	"context"
	"database/sql"
	"database/sql/driver"
	"os"
	"path/filepath"
	"testing"
	"time"

	gormiotdb "github.com/wkk778/gorm-iotdb"
	"gorm.io/gorm"
)

type noopConnPool struct{}

type telemetry struct {
	Time     time.Time `gorm:"column:time;iotdb:time"`
	Region   string    `gorm:"column:region;iotdb:tag"`
	DeviceID string    `gorm:"column:device_id;iotdb:tag"`
	Temp     float64   `gorm:"column:temp"`
}

func (noopConnPool) PrepareContext(_ context.Context, _ string) (*sql.Stmt, error) { return nil, nil }
func (noopConnPool) ExecContext(_ context.Context, _ string, _ ...interface{}) (sql.Result, error) {
	return driver.RowsAffected(0), nil
}
func (noopConnPool) QueryContext(_ context.Context, _ string, _ ...interface{}) (*sql.Rows, error) {
	return nil, nil
}
func (noopConnPool) QueryRowContext(_ context.Context, _ string, _ ...interface{}) *sql.Row {
	return nil
}

func TestOpenDryRun(t *testing.T) {
	db, err := gorm.Open(gormiotdb.New(gormiotdb.Config{Conn: noopConnPool{}}), &gorm.Config{DryRun: true, DisableAutomaticPing: true})
	if err != nil {
		t.Fatal(err)
	}

	sql := db.ToSQL(func(tx *gorm.DB) *gorm.DB {
		return tx.Create(&telemetry{Time: time.Unix(0, 0), Region: "cn", DeviceID: "d1", Temp: 20.5})
	})
	if sql == "" {
		t.Fatal("expected generated SQL")
	}
}

func TestOfficialSuiteLayout(t *testing.T) {
	root := filepath.Join("..", "testdata", "gorm-test")
	if _, err := os.Stat(root); err != nil {
		t.Fatalf("expected placeholder upstream suite directory: %v", err)
	}
}
