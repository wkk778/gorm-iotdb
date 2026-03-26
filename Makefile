SHELL := /bin/sh

.PHONY: test lint bench doc release

test:
	go test ./...
	go test ./tests/...

lint:
	golangci-lint run ./...

bench:
	bash ./scripts/update-benchmark.sh

doc:
	go test ./... -run TestDoesNotExist

release:
	goreleaser release --clean
