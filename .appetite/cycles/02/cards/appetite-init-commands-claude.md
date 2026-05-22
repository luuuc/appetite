---
slug: appetite-init-commands-claude
pitch: 02-02-slash-commands-dogfood
cycle: "02"
hill: uphill
progress: 0
hill_updated_at: 2026-05-22T18:11:39Z
---
extends `internal/cli/init.go` to accept `--commands <tool>` (only `claude` this cycle). Copies `commands/*.md` into `.claude/commands/`. Idempotent on identical content; refuses on diverged content without `--force`. Exit 2 on unknown tool name.
