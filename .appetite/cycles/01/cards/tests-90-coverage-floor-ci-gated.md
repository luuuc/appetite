---
slug: tests-90-coverage-floor-ci-gated
pitch: 01-02-workflow-engine-cli
cycle: "01"
hill: downhill
progress: 100
hill_updated_at: 2026-05-22T16:21:47Z
---
table-driven unit tests for every workflow operation covering happy path + every documented exit-2 violation, plus end-to-end subprocess tests walking the full loop. `make test` runs `go test -coverprofile=coverage.out ./...` plus a `tools/check-coverage` step that fails CI if any file or any function in `internal/workflow/`, `internal/cli/`, or `cmd/appetite/` reports coverage below 90.0%.
