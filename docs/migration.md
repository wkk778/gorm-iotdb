# Migration Guide

## From the original in-tree prototype

1. Replace `github.com/apache/iotdb-client-go/v2/gormiotdb` with `github.com/yourname/gorm-iotdb`.
2. Keep the same `gorm.Open(gormiotdb.Open(dsn), ...)` call shape.
3. Move any custom schema logic to `gorm:"...;iotdb:tag"` and `gorm:"...;iotdb:time"` tags.
4. If you already manage a shared connection pool, switch to `gormiotdb.New(gormiotdb.Config{Conn: pool})`.

## Planned 0.x to 1.0 work

- replace the placeholder upstream suite directory with the real GORM upstream test checkout
- harden migrator DDL against actual IoTDB parser differences across 1.1 and 1.2
- publish benchmark numbers for the nightly TPC-TS run
