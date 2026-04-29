# Benchmark

## Current State

This workspace does not contain a running IoTDB benchmark environment, so the numbers below are placeholders for the nightly GitHub Actions workflow.

| Date | Go | OS | IoTDB | Dataset | Write rows/s | Query p95 ms |
| --- | --- | --- | --- | --- | ---: | ---: |
| 2026-04-29 | 1.21 | ubuntu-latest | 1.2.0 | TPC-TS 1GB | pending | pending |

## Workflow

- `scripts/update-benchmark.sh` is the CI entrypoint.
- The nightly workflow is defined in `.github/workflows/nightly-benchmark.yml`.
- Results should be committed back into this file by the CI bot.
