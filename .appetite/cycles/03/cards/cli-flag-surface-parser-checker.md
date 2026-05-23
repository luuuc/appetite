---
slug: cli-flag-surface-parser-checker
pitch: 03-01-doc-drift-checks-as-test-targets
cycle: "03"
hill: uphill
progress: 0
hill_updated_at: 2026-05-23T09:37:40Z
---
extract `appetite <cmd> [flags]` from fenced shell blocks in `.doc/definition/07-mcp-and-cli.md`, diff against `appetite <cmd> -h`. Report `doc:line` ↔ binary divergence.
