#!/usr/bin/env sh
# Build + vet + full test suite + smoke. Exits non-zero on any failure.
set -eu
echo "==> go vet"
go vet ./...
echo "==> go build"
go build ./...
echo "==> go test (unit + Docker integration when a daemon is present)"
go test ./...
echo "==> build binary"
mkdir -p dist
go build -ldflags "-s -w" -o dist/sandbox-gate ./cmd/sandbox-gate
echo "==> smoke"
./dist/sandbox-gate version >/dev/null
echo "All checks passed"
