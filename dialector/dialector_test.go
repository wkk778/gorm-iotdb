package dialector

import (
	"context"
	"database/sql"
	"database/sql/driver"
	"sync"
	"testing"
	"time"

	"gorm.io/gorm"
	"gorm.io/gorm/schema"
)

type testTelemetry struct {
	Time     time.Time `gorm:"column:time;iotdb:time"`
	Region   string    `gorm:"column:region;iotdb:tag"`
	DeviceID string    `gorm:"column:device_id;iotdb:tag"`
	Temp     float64   `gorm:"column:temp"`
}

type noopConnPool struct{}

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

func TestDataTypeOf(t *testing.T) {
	var cache sync.Map
	s, err := schema.Parse(&testTelemetry{}, &cache, schema.NamingStrategy{})
	if err != nil {
		t.Fatal(err)
	}

	d := Dialector{}
	if got := d.DataTypeOf(s.LookUpField("Temp")); got != "DOUBLE" {
		t.Fatalf("expected DOUBLE, got %s", got)
	}
	if got := d.DataTypeOf(s.LookUpField("Time")); got != "TIMESTAMP" {
		t.Fatalf("expected TIMESTAMP, got %s", got)
	}
}

func TestGroupByShard(t *testing.T) {
	d := Dialector{config: Config{Conn: noopConnPool{}, TagShardFunc: func(table string, tags map[string]any) string {
		return table + "_" + tags["region"].(string)
	}}}

	db, err := gorm.Open(d, &gorm.Config{DryRun: true, DisableAutomaticPing: true})
	if err != nil {
		t.Fatal(err)
	}

	stmt := &gorm.Statement{DB: db}
	if err := stmt.Parse(&testTelemetry{}); err != nil {
		t.Fatal(err)
	}
	stmt.Dest = []testTelemetry{{Region: "cn"}, {Region: "us"}}
	stmt.Table = stmt.Schema.Table

	groups, err := d.groupByShard(stmt)
	if err != nil {
		t.Fatal(err)
	}
	if len(groups) != 2 {
		t.Fatalf("expected 2 groups, got %d", len(groups))
	}
}
