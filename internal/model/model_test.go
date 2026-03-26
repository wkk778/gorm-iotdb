package model

import (
	"reflect"
	"sync"
	"testing"
	"time"

	"gorm.io/gorm/schema"
)

type telemetry struct {
	Time     time.Time `gorm:"column:time;iotdb:time"`
	Region   string    `gorm:"column:region;iotdb:tag"`
	DeviceID string    `gorm:"column:device_id;iotdb:tag"`
	Temp     float64   `gorm:"column:temp"`
}

func TestParseColumns(t *testing.T) {
	var cache sync.Map
	s, err := schema.Parse(&telemetry{}, &cache, schema.NamingStrategy{})
	if err != nil {
		t.Fatal(err)
	}

	columns := ParseColumns(s)
	if len(columns) != 4 {
		t.Fatalf("expected 4 columns, got %d", len(columns))
	}
}

func TestTagValueMap(t *testing.T) {
	var cache sync.Map
	s, err := schema.Parse(&telemetry{}, &cache, schema.NamingStrategy{})
	if err != nil {
		t.Fatal(err)
	}

	values := TagValueMap(s, reflect.ValueOf(telemetry{Region: "cn", DeviceID: "d1"}))
	if values["region"] != "cn" {
		t.Fatalf("expected region cn, got %v", values["region"])
	}
}
