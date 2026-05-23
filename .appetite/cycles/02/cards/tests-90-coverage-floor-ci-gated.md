---
slug: tests-90-coverage-floor-ci-gated
pitch: 02-01-mcp-server
cycle: "02"
hill: downhill
progress: 100
hill_updated_at: 2026-05-22T18:48:56Z
---
table-driven handler tests covering happy path + every documented error code for every tool. End-to-end test in `internal/mcp/e2e_test.go` walks the full loop via JSON-RPC (`init` → `signal` → `shape` → `cycle_new` → `bet` → `cut` → `hill` → `cooldown`). `tools/check-coverage` extends its scope to `internal/mcp/`. Sub-90% files do not merge.
