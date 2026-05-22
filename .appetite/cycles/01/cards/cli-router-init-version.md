---
slug: cli-router-init-version
pitch: 01-02-workflow-engine-cli
cycle: "01"
hill: downhill
progress: 100
hill_updated_at: 2026-05-22T16:21:47Z
---
`internal/cli/` with subcommand dispatch using stdlib `flag`. Top-level flags: `--dir`, `-h`. `appetite init` creates the skeleton directories (idempotent). `appetite version` already works; route it through the new router.
