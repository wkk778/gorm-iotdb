# gorm-iotdb

[![CI](https://github.com/wkk778/gorm-iotdb/actions/workflows/ci.yml/badge.svg)](https://github.com/wkk778/gorm-iotdb/actions/workflows/ci.yml)
[![Release](https://github.com/wkk778/gorm-iotdb/actions/workflows/release.yml/badge.svg)](https://github.com/wkk778/gorm-iotdb/actions/workflows/release.yml)
[![pkg.go.dev](https://pkg.go.dev/badge/github.com/wkk778/gorm-iotdb.svg)](https://pkg.go.dev/github.com/wkk778/gorm-iotdb)

`gorm-iotdb` is a standalone GORM v1.25.x dialector for Apache IoTDB built from the original `gormiotdb/dialector.go` prototype and expanded into a reusable module with a dedicated driver wrapper, migrator, examples, and CI scaffolding.

## Status

The source tree in this workspace is organized as a repo-ready module under this directory. The exported copy is independent from the parent workspace and uses the final module path `github.com/wkk778/gorm-iotdb`.

The official GORM upstream suite integration is wired as a repository layout and script contract, but the actual upstream submodule content still needs to be fetched in a network-enabled clone.

## Install

```bash
go get github.com/wkk778/gorm-iotdb
```

Repository: `https://github.com/wkk778/gorm-iotdb`

## Quick Start

```go
package main

import (
    "log"
    "time"

    gormiotdb "github.com/wkk778/gorm-iotdb"
    "gorm.io/gorm"
)

type Telemetry struct {
    Time      time.Time `gorm:"column:time;iotdb:time"`
    Region    string    `gorm:"column:region;iotdb:tag"`
    DeviceID  string    `gorm:"column:device_id;iotdb:tag"`
    Temp      float64   `gorm:"column:temp"`
    Humidity  float64   `gorm:"column:humidity"`
}

func main() {
    db, err := gorm.Open(gormiotdb.Open("iotdb://127.0.0.1:6667?username=root&password=root"), &gorm.Config{})
    if err != nil {
        log.Fatal(err)
    }

    if err := db.AutoMigrate(&Telemetry{}); err != nil {
        log.Fatal(err)
    }
}
```

## Repository Layout

- `dialector/`: core GORM dialector and migrator implementation
- `driver/`: `database/sql` wrapper around the upstream IoTDB client
- `internal/`: private schema-mapping helpers
- `examples/`: runnable usage samples
- `tests/`: dry-run tests and upstream suite adapter entrypoint
- `docs/`: design, benchmark, and migration notes
- `scripts/`: helper scripts for export, tests, and benchmark updates

## Configuration

| Field | Description | Default |
| --- | --- | --- |
| `DSN` | IoTDB connection string | required |
| `DriverName` | `database/sql` driver name | `iotdb` |
| `Conn` | prebuilt `gorm.ConnPool` | `nil` |
| `TagShardFunc` | resolves physical table by tag set | disabled |

## Tag Model Mapping

| GORM tag | IoTDB meaning |
| --- | --- |
| `iotdb:time` | time column |
| `iotdb:tag` | tag column |
| omitted | measurement field |
| `type:...` | explicit IoTDB type override |

## Performance Tuning Checklist

- Keep `CreateBatchSize` aligned with tablet size used by the upstream IoTDB client.
- Use `TagShardFunc` when high-cardinality tags create hot logical tables.
- Reuse a shared `*sql.DB` pool through `Config.Conn` in long-running services.
- Prefer server-side time and tag predicates to reduce scan volume.
- Run the nightly benchmark workflow after changing write-path code.

## Examples

- [`examples/crud`](./examples/crud)
- [`examples/batch`](./examples/batch)
- [`examples/ts-query`](./examples/ts-query)
- [`examples/tag-filter`](./examples/tag-filter)
- [`examples/rest-ts`](./examples/rest-ts)

## Tests

```bash
make test
make lint
make bench
make release
```

The `tests/` directory is ready for the upstream GORM test harness. After initializing the submodule, run:

```bash
git submodule update --init --recursive
go test ./tests/...
```

## Releases

- Push a tag like `v0.1.0` to trigger the release workflow.
- GitHub Actions uses `.goreleaser.yml` to generate release notes, checksums, and archives.
- Current archive targets are `linux-amd64` and `darwin-arm64`.

## FAQ

### Does this already pass the whole upstream GORM suite?

Not in this offline workspace. The repository contains the adapter entrypoint, docker-compose file, CI workflow, and helper scripts, but the real upstream suite must be fetched into `testdata/gorm-test/` first.

### Does IoTDB support every relational GORM feature?

No. Associations, foreign keys, relational indexes, and some transactional semantics are database-specific. The dialector exposes best-effort no-op migrator behavior for unsupported features so GORM integration remains predictable.

### Which IoTDB versions are targeted?

The repository is structured for IoTDB `1.1.x`, `1.2.x`, and forward-compatible type mapping for `2.x` additions.

## Export Notes

Run `go mod tidy` in a network-enabled environment before the first push if the upstream IoTDB client module is not already cached.
