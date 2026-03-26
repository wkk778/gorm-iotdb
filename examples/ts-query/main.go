package main

import (
	"fmt"
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

	db, err := gorm.Open(gormiotdb.Open(dsn), &gorm.Config{})
	if err != nil {
		log.Fatal(err)
	}

	var rows []Telemetry
	if err := db.Where("time >= ?", time.Now().Add(-time.Hour)).Order("time desc").Limit(10).Find(&rows).Error; err != nil {
		log.Fatal(err)
	}
	fmt.Println(len(rows))
}
