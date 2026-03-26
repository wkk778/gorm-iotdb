# Changelog

## [0.1.0] - 2026-03-26

- bootstrapped a standalone Go module at `github.com/wkk778/gorm-iotdb`
- migrated the original `gormiotdb/dialector.go` entrypoints into `dialector/` and a root re-export
- added a reusable IoTDB `database/sql` wrapper in `driver/`
- added a best-effort IoTDB migrator, tag-based shard routing, and dry-run-safe tests
- added example programs, docs, CI scaffolding, docker-compose, and Makefile targets
