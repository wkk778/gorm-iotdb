package main

import (
	"log"
	"os"
	"time"

	gormiotdb "github.com/wkk778/gorm-iotdb"
	"gorm.io/gorm"
)

type Telemetry struct {
	Time     time.Time `gorm:"column:time;iotdb:time"`
	Region   string    `gorm:"column:region;iotdb:tag"`
	DeviceID string    `gorm:"column:device_id;iotdb:tag"`
	Temp     float64   `gorm:"column:temp"`
}

func main() {
	dsn := os.Getenv("IOTDB_DSN")
	if dsn == "" {
		dsn = "iotdb://127.0.0.1:6667?username=root&password=root"
	}

	db, err := gorm.Open(gormiotdb.New(gormiotdb.Config{
		DSN: dsn,
		TagShardFunc: func(table string, tags map[string]any) string {
			return table + "_" + tags["region"].(string)
		},
	}), &gorm.Config{CreateBatchSize: 100})
	if err != nil {
		log.Fatal(err)
	}

	batch := []Telemetry{{Time: time.Now(), Region: "cn", DeviceID: "d1", Temp: 20.5}, {Time: time.Now(), Region: "us", DeviceID: "d2", Temp: 21.5}}
	if err := db.Create(&batch).Error; err != nil {
		log.Fatal(err)
	}
}
