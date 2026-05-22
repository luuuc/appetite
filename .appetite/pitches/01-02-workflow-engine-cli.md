---
slug: 01-02-workflow-engine-cli
title: Workflow Engine + CLI
appetite: large
status: shipped
shaped_at: 2026-05-22T16:19:14Z
---
## Problem

Bootstrap (01-01) shipped a Go module with entity types, a `Store` interface, a Markdown/YAML adapter, and a state-machine validator — but nothing operates on them. `cmd/appetite/main.go` is still a stub that prints "not yet implemented." There is no way to run the Shape Up loop. You cannot `appetite init` a folder, draft a pitch, open a cycle, place a bet, cut cards, update hill positions, ship, or cool down. The product, as it stands, is plumbing without a tap.

Until the loop runs end-to-end from the terminal, Appetite cannot be dogfooded — and a Shape Up workflow tool that hasn't been used to ship itself is not a real product yet.

## Appetite

3 real days. This is the walking skeleton — the thinnest end-to-end vertical that lets a single operator drive the entire Shape Up loop through one surface (the CLI). 3 days is enough to wire the workflow package on top of the existing store, implement the eight loop commands, render a usable `status` view, and stop. Anything beyond — JSON output, `doctor`, sweep commands, fancy color rendering, MCP, sync — exceeds this appetite and gets re-shaped into a later cycle.

If 01-02 grows past 3 days, it's a shaping failure. The corrective is to cut commands (defer non-loop reads to a later cycle), not to extend.

## Solution

A workflow package sits between the existing storage layer and a new CLI router. The workflow package is the only place state-machine rules live; the CLI is a thin translator from argv to workflow calls.

```
cmd/appetite/main.go
        │
        ▼
internal/cli/          ── argv parsing, subcommand routing, output formatting
        │
        ▼
internal/workflow/     ── operations: Shape, Finalize, OpenCycle, Bet, Pass, Cut, Hill, Cooldown
        │                 enforces transitions via internal/model/transitions.go
        ▼
internal/store/        ── existing interface
internal/markdown/     ── existing Markdown/YAML adapter
```

Every CLI command is a function in `internal/workflow/` that takes a `Store` and a request struct, returns a result struct and a typed error. The CLI layer adds argv parsing and human-readable output, nothing more. This split keeps the workflow package callable from the future MCP server without rework.

### Loop commands in scope

The eight commands that complete one full Shape Up loop, per `.doc/definition/07-mcp-and-cli.md`:

| Command | What it does |
|---|---|
| `appetite init` | Create `.appetite/{signals/raw,pitches,cycles}/`. Idempotent. |
| `appetite signal add <text>` | Write a signal file. |
| `appetite shape --new\|--from\|--finalize` | Create / seed / finalize a pitch. |
| `appetite cycle new <id> --appetite <duration>` | Open a cycle in `building`. |
| `appetite bet <pitch> --cycle <id> --appetite <size>` | Place a bet. Validates pitch is `shaped` and cycle is `building`. |
| `appetite pass <pitch> --reason <text>` | Mark a shaped pitch as `passed`. |
| `appetite cut <pitch>` | Read pitch's `## Scope`, write cards under the active cycle. |
| `appetite hill <card> --position <p> --progress <n> [--done]` | Update hill state. `--done` required for progress=100. |
| `appetite cooldown [--days <n>] [close]` | Open or close cooldown. |
| `appetite status` | Render the current cycle: bets, hill positions, stuck warnings. |

### Status rendering

Text only. The format from `.doc/definition/07-mcp-and-cli.md`:

```
Cycle 2026-w15 (5d, day 3/5, status: building)

Bets:
  [==>       ]  csv-export        medium    2/3 cards down
  [======>   ]  hill-chart-mvp    small     1/1 cards down

Passed: dark-mode (not worth it this cycle)
```

Color is nice-to-have, not required. Width is fixed at 80 columns.

### Stuck detection

Read-time computation in `internal/workflow/status.go`: a card is "stuck" if `hill == uphill` and `now - hill_updated_at > 0.5 * cycle.appetite`. Surface as `⚠` next to the card in the status view. Advisory only — does not block any transition.

### Exit codes

Per `.doc/definition/07-mcp-and-cli.md`:

| Code | Meaning |
|---|---|
| `0` | Success |
| `1` | Generic error |
| `2` | Workflow violation (skipped state, missing `--done` on progress=100, etc.) |
| `3` | Not found |

Exit codes are the only stable machine surface in 01-02. JSON output is deferred.

## Rabbit holes

- **CLI framework choice.** Don't pull in `cobra`/`urfave-cli`. Standard library `flag` plus a small subcommand router (mirrors the sibling repos' approach). Adding a framework now means living with it forever.
- **Pretty status output.** Easy to spend a day on color, Unicode box-drawing, terminal-width detection. The 80-column fixed-width view in `.doc/definition/07-mcp-and-cli.md` is the target — match it, then stop.
- **Parsing the pitch `## Scope` section.** `cut` needs to read scope cards from the pitch body. Keep the parser strict: lines matching `^- \[ \] \*\*<title>\*\* — <desc>$` are cards, everything else is ignored. Don't try to be clever about variant markdown.
- **The workflow package becoming a god object.** One file per command (`shape.go`, `bet.go`, `cut.go`, …), each ~50 lines. If one grows beyond 100, the validation should likely move to `internal/model/transitions.go`.
- **90% coverage floor is real budget.** Writing tests *to* the floor (not just past it) is roughly a third of the appetite. Don't discover this on day 3. Write tests alongside each command card, not as a final batch. If a function is hard to cover, that's a design signal — refactor, don't lower the gate.
- **`go tool cover` quirks.** Per-function coverage is reported by `go tool cover -func`, not by `-cover` alone. The CI gate parses that output. Coverage on `main.go` is achieved via subprocess tests in `cmd/appetite/main_test.go`, not by importing `main`.

## No-gos

- **No MCP server.** That is a later cycle. The workflow package is designed to be wrapped later — do not pre-build the wrapping.
- **No `--json` flag.** Defer to a later cycle (the MCP layer needs structured output, the CLI does not yet).
- **No `appetite doctor`.** Validation lives in the workflow package's normal write path; a separate validator command is a Cycle 2 concern.
- **No sweep commands** (`signal/pitch/cycle sweep`). Closed cycles and stale signals stay put for v0.1; sweeps land with the cleanup polish in a later cycle.
- **No sync.** Cycle 2+.
- **No hill-chart visualization beyond the inline progress bar.** No SVG, no separate chart command. The status text view is enough.
- **No config loader.** Defaults are hardcoded for v0.1 (`cycle_appetite: 5d`, `cooldown: 1d`).

## Scope

- [x] **Workflow package skeleton** — `internal/workflow/` with one file per loop operation (`shape.go`, `bet.go`, `cut.go`, `hill.go`, `cooldown.go`, `cycle.go`, `pass.go`, `signal.go`, `status.go`). Each exposes a `func Xxx(s store.Store, req XxxRequest) (XxxResult, error)`. Typed errors: `ErrInvalidTransition`, `ErrNotFound`, `ErrDoneCriteriaUnmet`.
- [x] **CLI router + `init` + `version`** — `internal/cli/` with subcommand dispatch using stdlib `flag`. Top-level flags: `--dir`, `-h`. `appetite init` creates the skeleton directories (idempotent). `appetite version` already works; route it through the new router.
- [x] **Signal + shape commands** — `appetite signal add`, `appetite shape --new`, `appetite shape --from <signal>`, `appetite shape --finalize <slug>`. Finalize validates the five ingredients and at least one scope card before flipping status to `shaped`; otherwise exits 2 with a precise message.
- [x] **Cycle + bet + pass commands** — `appetite cycle new <id> --appetite <duration>`, `appetite bet <pitch> --cycle <id> --appetite <size>`, `appetite pass <pitch> --reason <text>`. Bet validates the pitch is `shaped` and an active cycle in `building` exists; pass records a `passed_at` timestamp and reason in `cycle.yml`.
- [x] **Cut command + Scope parser** — `appetite cut <pitch>` reads the pitch's `## Scope` section via a strict line-by-line parser (regex `^- \[ \] \*\*<title>\*\* — <desc>$`), writes one card file per match under `.appetite/cycles/<id>/cards/<slug>.md` with `hill: uphill / progress: 0`, then flips the pitch from `bet` to `building`. Parser errors print the offending line and exit 2; partial writes are rolled back.
- [x] **Hill command** — `appetite hill <card> --position <uphill|downhill> --progress <0-99>` for in-flight updates; `appetite hill <card> --progress 100 --done` for ship. Without `--done`, progress=100 exits 2. Automatic pitch→`shipped` and cycle→`shipping` transitions fire when the last card ships.
- [x] **Status + cooldown commands** — `appetite status` renders the 80-column text view (cycle line, progress bars, passed list, stuck warnings). `appetite cooldown [--days <n>]` opens cooldown for the current cycle; `appetite cooldown close` closes it. Refuses to open while pitches are still `building`.
- [x] **Tests + 90% coverage floor (CI-gated)** — table-driven unit tests for every workflow operation (`internal/workflow/*_test.go`) covering happy path + every documented exit-2 violation, plus end-to-end subprocess tests in `cmd/appetite/main_test.go` walking the full loop (`init → signal → shape → cycle new → bet → cut → hill → cooldown`). `make test` includes `go test -coverprofile=coverage.out ./...` and a `tools/check-coverage` step that runs `go tool cover -func` and fails CI if **any file or any function** in `internal/workflow/`, `internal/cli/`, or `cmd/appetite/` reports coverage below 90.0%. The 90% floor is non-negotiable for v0.1 and forward; sub-90% files do not merge.

---

## Reference

| Convention | Source |
|---|---|
| State machine | `internal/model/transitions.go` (shipped in 01-01) |
| Storage interface | `internal/store/store.go` (shipped in 01-01) |
| CLI style | stdlib `flag` + subcommand router, mirroring brain/beacon |
| Output format | `.doc/definition/07-mcp-and-cli.md` |
| Loop semantics | `.doc/definition/03-workflow.md` |
