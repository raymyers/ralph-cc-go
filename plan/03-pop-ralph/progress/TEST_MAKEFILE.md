# TEST_MAKEFILE Progress

## Task
Split `make test` to be non-verbose and have slow tests in `make test-slow`, `make check` does all.

## Analysis
- Previous: `go test -v ./...` (verbose, all tests, ~13s uncached)
- Slow tests: `TestE2ERuntimeYAML` in `cmd/ralph-cc/integration_test.go` takes ~12s (66 runtime tests involving assemble/link/run)
- Other tests are fast (<1s per package)

## Implementation
1. Changed `make test` to `go test -skip 'TestE2ERuntimeYAML' ./...` (~3.4s)
2. Added `make test-slow` running `go test -run 'TestE2ERuntimeYAML' ./...` (~12s)
3. Added `make test-all` target running both
4. Updated `check` to use `test-all`

## Results
- `make test`: ~3.4s (fast, non-verbose)
- `make test-slow`: ~12s (runtime tests)
- `make check`: passes (lint + test-all)

## Status: DONE
