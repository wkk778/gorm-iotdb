#!/usr/bin/env bash
set -euo pipefail

echo "initializing upstream gorm suite"
git submodule update --init --recursive

echo "running local adapter tests"
go test ./tests/...
