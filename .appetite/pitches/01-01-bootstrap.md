---
slug: 01-01-bootstrap
title: Bootstrap
appetite: medium
status: shipped
shaped_at: 2026-05-22T16:19:14Z
---
## Problem

Appetite has definition docs but zero code. Before anything can be built — workflow engine, CLI, MCP, sync — there needs to be a Go module that compiles, a storage interface that the rest of the system programs against, and a Markdown/YAML adapter that can read and write `.appetite/` files.

Without this foundation, every subsequent pitch starts by also building the scaffolding it stands on. That's how pitches bloat.

## Appetite

2 real days. This is scaffolding — it should be boring and correct. We're not willing to spend more than 2 days because the output is a skeleton, not a feature. If it takes longer, the abstraction is too fancy.

## Solution

A Go module (`github.com/luuuc/appetite`) with three moving parts, mirroring Brain's bootstrap shape:

1. **Storage interface** — a Go interface that defines read, write, list, and delete for the core entities (signals, pitches, cycles, cards, cooldowns). One interface, many entity types. Sentinel errors (`ErrNotFound`, `ErrInvalidTransition`) for callers to distinguish.
2. **Markdown/YAML adapter** — implements the storage interface against a `.appetite/` directory. Pitches and signals and cards are `.md` with YAML frontmatter. Cycles and cooldowns are `.yml`. Atomic write-to-temp + rename. Auto-creates subdirectories.
3. **Entity types** — Go structs for Signal, Pitch, Cycle, Card, Cooldown, Bet. YAML struct tags. Shared `Status` enum per entity. A `Transition` validator that encodes the state machine from `.doc/definition/03-workflow.md`.

### Directory layout

```
appetite/
  go.mod
  go.sum
  .goreleaser.yml
  .golangci.yml
  Makefile
  cmd/
    appetite/
      main.go
  internal/
    version/
      version.go
    model/
      signal.go
      pitch.go
      cycle.go
      card.go
      cooldown.go
      bet.go
      transitions.go
    store/
      store.go
      conformance.go
    markdown/
      adapter.go
      adapter_test.go
```

### No-gos for this pitch

- No CLI commands — that's 01-02
- No MCP server — that's a later cycle
- No sync engine — that's a later cycle
- No hill chart computation — that's the read path pitch
- No LLM calls — never
- No PostgreSQL — not this year

## Rabbit holes

- **YAML vs. Markdown for cycles.** Cycles are state-heavy, pitches are narrative. Use YAML for `cycle.yml` and `cooldown.yml`, Markdown for pitches/signals/cards. Don't try to force one format.
- **State machine scope.** The `transitions.go` validator encodes the rules from `.doc/definition/03-workflow.md`. Keep it strict — adding a skipped state is not a bug fix, it's a design change.
- **Naming collisions with shape-cli / the-craft-workflow.** Brand is `appetite`, binary is `appetite`, module is `github.com/luuuc/appetite`. No references to prior tools in the code or commits — they're prior art, not imports.

## No-gos

- No CLI — see above
- No MCP server — see above
- No sync adapter — see above
- No hill chart stuck detection — that's a read-time computation, lives in the read path pitch
- No `appetite.yml` loader — hardcode defaults

## Scope

- [x] **Go module init** — `go mod init github.com/luuuc/appetite`, Go 1.25.5, `cmd/appetite/`, `internal/`, `go vet` and `go test` pass
- [x] **Build scaffolding** — `.goreleaser.yml`, `Makefile` (build, test, lint, clean), `.golangci.yml`
- [x] **CI pipeline** — `.github/workflows/ci.yml` (test + lint on push/PR), `.github/workflows/release.yml` (goreleaser on tag), `.github/workflows/smoke.yml` (post-release binary smoke test)
- [x] **Version package** — `internal/version/version.go` with ldflags injection
- [x] **Entity structs** — Signal, Pitch, Cycle, Card, Cooldown, Bet with YAML tags and Status enums
- [x] **Transitions validator** — pure function `ValidateTransition(from, to Status) error` encoding the state machine from `.doc/definition/03-workflow.md`
- [x] **Storage interface** — `Store` interface: `Write`, `Read`, `List`, `Delete` plus entity-typed helpers. `Filter` struct for push-down filtering.
- [x] **Conformance test suite** — `internal/storetest/conformance.go` validating any `Store` implementation (kept outside `internal/store/` so `testing` stays out of the production import graph — matches brain's pattern)
- [x] **Markdown adapter: read/write/list/delete** — atomic writes, frontmatter parsing, directory auto-creation
- [x] **Markdown adapter tests** — run conformance + markdown-specific tests (t.TempDir, frontmatter edge cases)

---

## Reference

Appetite follows the same project conventions as its sibling repos (brain, beacon, council). See their `CLAUDE.md` and `.doc/pitches/01-01-bootstrap.md` for exact patterns:

| Convention | Pattern | Reference |
|---|---|---|
| **Go version** | 1.25.5, `go-version-file: go.mod` in CI | `brain/go.mod` |
| **Module path** | `github.com/luuuc/appetite` | consistent with siblings |
| **Dependencies** | Minimal — `gopkg.in/yaml.v3` only at bootstrap | `brain/go.mod` |
| **Storage interface** | Interface + adapter factory + conformance test suite | `brain/internal/store/`, `beacon/internal/beacondb/` |
| **Version injection** | `internal/version/version.go` + ldflags | `brain/internal/version/version.go` |
| **Build** | GoReleaser + Makefile | `brain/.goreleaser.yml`, `brain/Makefile` |
| **Linting** | `.golangci.yml` v2 with errcheck + staticcheck | `brain/.golangci.yml` |
| **CI** | ci.yml + release.yml + smoke.yml | `brain/.github/workflows/` |
| **Testing** | stdlib `testing`, `t.TempDir()`, table-driven | `brain/internal/markdown/` |
| **Binary entry point** | `cmd/appetite/main.go` → `internal/cmd` | `brain/cmd/brain/main.go` |

---

*Prior-art note: 01-01 shipped before Appetite could track itself. It is recorded here as `shipped` for completeness; it was not part of Cycle 1's bets (`.appetite/cycles/01/cycle.yml`).*
