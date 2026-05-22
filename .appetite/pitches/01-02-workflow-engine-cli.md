---
slug: 01-02-workflow-engine-cli
title: Workflow Engine + CLI
appetite: ""
status: shipped
shaped_at: 2026-05-22T16:19:14Z
---
## Problem

Bootstrap (01-01) shipped a Go module with entity types, a `Store` interface, a Markdown/YAML adapter, and a state-machine validator — but nothing operates on them. `cmd/appetite/main.go` is still a stub that prints "not yet implemented." There is no way to run the Shape Up loop. You cannot `appetite init` a folder, draft a pitch, open a cycle, place a bet, cut cards, update hill positions, ship, or cool down. The product, as it stands, is plumbing without a tap.

Until the loop runs end-to-end from the terminal, Appetite cannot be dogfooded — and a Shape Up workflow tool that hasn't been used to ship itself is not a real product yet.

## Appetite

3 real days. This is the walking skeleton — the thinnest end-to-end vertical that lets a single operator drive the entire Shape Up loop through one surface (the CLI). 3 days is enough to wire the workflow package on top of the existing store, implement the eight loop commands, render a usable `status` view, and stop. Anything beyond — JSON output, `doctor`, sweep commands, fancy color rendering, MCP, sync — exceeds this appetite and gets re-shaped into a later cycle.

If 01-02 grows past 3 days, it's a shaping failure. The corrective is to cut commands (defer non-loop reads to 01-04), not to extend.

## Solution

See `.doc/pitches/01-02-workflow-engine-cli.md` for the full solution narrative. The workflow package sits between storage and a thin CLI router; the eight loop commands (`init`, `signal`, `shape`, `cycle`, `bet`, `pass`, `cut`, `hill`, `cooldown`, `status`) each map to one workflow function. Exit codes per `.doc/definition/07-mcp-and-cli.md` (0/1/2/3).

## Rabbit holes

See `.doc/pitches/01-02-workflow-engine-cli.md`: CLI framework choice (stdlib only), pretty status output, strict scope parser, workflow-package-as-god-object, 90% coverage floor as real budget, `go tool cover` quirks.

## No-gos

No MCP server, no `--json` flag, no `appetite doctor`, no sweep commands, no sync, no hill-chart visualization beyond inline progress bar, no config loader.

## Scope

- [ ] **Workflow package skeleton** — `internal/workflow/` with one file per loop operation (`shape.go`, `bet.go`, `cut.go`, `hill.go`, `cooldown.go`, `cycle.go`, `pass.go`, `signal.go`, `status.go`). Each exposes a `func Xxx(s store.Store, req XxxRequest) (XxxResult, error)`. Typed errors: `ErrInvalidTransition`, `ErrNotFound`, `ErrDoneCriteriaUnmet`.
- [ ] **CLI router + `init` + `version`** — `internal/cli/` with subcommand dispatch using stdlib `flag`. Top-level flags: `--dir`, `-h`. `appetite init` creates the skeleton directories (idempotent). `appetite version` already works; route it through the new router.
- [ ] **Signal + shape commands** — `appetite signal add`, `appetite shape --new`, `appetite shape --from <signal>`, `appetite shape --finalize <slug>`. Finalize validates the five ingredients and at least one scope card before flipping status to `shaped`; otherwise exits 2 with a precise message.
- [ ] **Cycle + bet + pass commands** — `appetite cycle new <id> --appetite <duration>`, `appetite bet <pitch> --cycle <id> --appetite <size>`, `appetite pass <pitch> --reason <text>`. Bet validates the pitch is `shaped` and an active cycle in `building` exists; pass records a `passed_at` timestamp and reason in `cycle.yml`.
- [ ] **Cut command + Scope parser** — `appetite cut <pitch>` reads the pitch's `## Scope` section via a strict line-by-line parser, writes one card file per match under `.appetite/cycles/<id>/cards/<slug>.md` with `hill: uphill / progress: 0`, then flips the pitch from `bet` to `building`. Parser errors print the offending line and exit 2; partial writes are rolled back.
- [ ] **Hill command** — `appetite hill <card> --position <uphill|downhill> --progress <0-99>` for in-flight updates; `appetite hill <card> --progress 100 --done` for ship. Without `--done`, progress=100 exits 2. Automatic pitch→`shipped` and cycle→`shipping` transitions fire when the last card ships.
- [ ] **Status + cooldown commands** — `appetite status` renders the 80-column text view (cycle line, progress bars, passed list, stuck warnings). `appetite cooldown [--days <n>]` opens cooldown for the current cycle; `appetite cooldown close` closes it. Refuses to open while pitches are still `building`.
- [ ] **Tests + 90% coverage floor (CI-gated)** — table-driven unit tests for every workflow operation covering happy path + every documented exit-2 violation, plus end-to-end subprocess tests walking the full loop. `make test` runs `go test -coverprofile=coverage.out ./...` plus a `tools/check-coverage` step that fails CI if any file or any function in `internal/workflow/`, `internal/cli/`, or `cmd/appetite/` reports coverage below 90.0%.
