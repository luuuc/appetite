---
slug: 02-02-slash-commands-dogfood
title: Slash Commands + Real-Cycle Dogfood
appetite: small
status: building
shaped_at: 2026-05-22T18:05:38Z
---
## Problem

`.doc/definition/08-ai-workflow.md` defines eight slash commands (`/signal`, `/shape`, `/open-cycle`, `/bet`, `/no-bet`, `/break-out`, `/cooldown`, `/cycle`) as the AI-native primary surface, and ships none. `.claude/commands/` carries `shape`, `craft`, `commit`, `pr`, `council` — workflow-adjacent helpers from the bootstrap, not the Appetite workflow commands. AI operators today have no installable surface for driving Appetite from Claude Code.

Cycle 1's dogfood was real but partial: `.appetite/signals/raw/` is empty. The `signal → shape` path was never exercised; no friction encountered during 01-02 or 01-03 was captured in flight. Cycle 2 needs to fix both — ship the slash commands AND use them to drive real signal capture during 02-01.

## Appetite

4 real hours for the implementation. The slash commands are markdown files plus an `init --commands` flag — the code is genuinely small. The harder budget is what 4 hours buys *after* implementation: enough live use of the surface during 02-01 to capture three real signals and walk one of them into a Cycle 3 pitch draft.

If 02-02 stretches because the dogfood "needs more time", that is a signal that 02-01 is in trouble, not 02-02. Cut, don't extend.

## Solution

Two thin parts, then the dogfood.

### 1. Reference slash commands under `commands/`

Eight markdown files at the repo root in `commands/`, mirroring the table in `.doc/definition/08-ai-workflow.md`:

| File | Stage | Calls |
|---|---|---|
| `signal.md` | Capture | `appetite_signal` |
| `shape.md` | Draft / finalize pitch | `appetite_shape` |
| `open-cycle.md` | Create cycle | `appetite_cycle_new` |
| `bet.md` | Bet (with or without slug) | `appetite_bet` |
| `no-bet.md` | Pass | `appetite_pass` |
| `break-out.md` | Cut | `appetite_cut` |
| `cooldown.md` | Open / close cooldown | `appetite_cooldown` |
| `cycle.md` | Status read | `appetite_status` |

Each command's body walks the operator through the conversation, calls the MCP tool only after explicit H approval for rule transitions, and prints the result. The existing `.claude/commands/shape.md` predates Appetite — keep it as a separate "draft pitch text" helper; the new `commands/shape.md` is the workflow command that calls `appetite_shape` MCP. Do not merge them — they have different jobs and will rot together if conflated.

### 2. `appetite init --commands claude`

Extends `init` to optionally copy `commands/*.md` into `.claude/commands/`. Idempotent. If a target file exists with identical content, skip; if it exists with different content, refuse without `--force` and print which file conflicts. Per `.doc/definition/08-ai-workflow.md`.

### 3. The dogfood (the second half of this pitch's value)

During Cycle 2:

- Every *rule transition* on 02-01 (`/open-cycle`, `/bet`, `/break-out`, `/cooldown`) happens through a slash command at least once. The CLI is allowed as a fallback for inline hill updates (per `08-ai-workflow.md`, `/hill` is intentionally not a slash command).
- `/signal` is exercised at least three times during 02-01 with real friction from building (e.g. "JSON-RPC framing spec choice", "Coverage gate scope unclear", "MCP server startup path noisy"). Signals land in `.appetite/signals/raw/`.
- At least one captured signal is walked through `/shape` (without finalize) during 02-02's own cooldown, producing a Cycle 3 draft pitch in `.appetite/pitches/03-NN-<slug>.md` with `status: shaping`. This is the proof that the loop closes signal → shaped without manual stitching.

## Rabbit holes

- **Slash command bloat.** Resist building `/hill` and `/ship`. Per `08-ai-workflow.md` those are inline mid-work moments; a slash command would force ceremony around what should be one MCP call. Eight commands is the limit.
- **Cursor / Codex parity.** Defer. Reference implementations target Claude Code only this cycle. `--commands cursor` and `--commands codex` land in a later pitch once the Claude Code shape is proven.
- **Reshaping mid-dogfood.** If a slash command's spec is wrong during use, file a signal — do not reshape this pitch. Same rule as 01-03.
- **Conflating bootstrap helpers with workflow commands.** `.claude/commands/shape.md` (bootstrap helper) and `commands/shape.md` (workflow command) have the same name and different jobs. Keep them separate.
- **Fake signals to hit the "≥3" gate.** The signal count card is a quality signal, not a count target. If only one real friction surfaces, ship with one and note the gap in cooldown — do not invent.

## No-gos

- **No `/hill` slash command.** Inline only.
- **No `/ship` slash command.** Inline only; ships via `appetite_hill --done`.
- **No Cursor or Codex command installers.** This cycle is Claude Code only.
- **No reshaping mid-dogfood.** Signals only.
- **No `commands/` content that duplicates definition docs.** Each command's markdown is short — link to `08-ai-workflow.md` for canonical semantics.

## Scope

- [ ] **Reference slash commands in `commands/`** — eight markdown files (`signal.md`, `shape.md`, `open-cycle.md`, `bet.md`, `no-bet.md`, `break-out.md`, `cooldown.md`, `cycle.md`). Each: frontmatter (allowed tools, brief description), short body walking the operator through the conversation, explicit H-approval gate for rule transitions, exit on error.
- [ ] **`appetite init --commands claude`** — extends `internal/cli/init.go` to accept `--commands <tool>` (only `claude` this cycle). Copies `commands/*.md` into `.claude/commands/`. Idempotent on identical content; refuses on diverged content without `--force`. Exit 2 on unknown tool name.
- [ ] **Live dogfood — slash-driven Cycle 2 transitions** — Cycle 2's bets are placed via `/bet`, cut via `/break-out`, cooled down via `/cooldown`. The `.appetite/cycles/02/cycle.yml` audit trail shows MCP-driven transitions; git history of `.claude/commands/` shows the install.
- [ ] **Live signal capture (≥3)** — `/signal` exercised at least three times during 02-01 with real frictions. Files land in `.appetite/signals/raw/`. Each signal has a one-line title and a short paragraph of context — no triage, no prioritization.
- [ ] **Cycle 3 pitch draft from a signal** — during 02-02's cooldown, walk one captured signal through `/shape` (no finalize) to produce `.appetite/pitches/03-NN-<slug>.md` with `status: shaping`.
- [ ] **CLAUDE.md rule block** — append the "Ready-to-Paste CLAUDE.md Rule Block" from `.doc/definition/08-ai-workflow.md` to `CLAUDE.md`. Locks in H-only rule transitions and inline hill updates for the AI agent during Cycle 2 and beyond.

---

## Reference

| Convention | Source |
|---|---|
| Slash command spec | `.doc/definition/08-ai-workflow.md` |
| MCP tools | `.doc/definition/07-mcp-and-cli.md` (shipped via 02-01) |
| Workflow loop semantics | `.doc/definition/03-workflow.md` |
| CLAUDE.md rule block | `.doc/definition/08-ai-workflow.md` (bottom) |
