---
slug: 02-01-mcp-server
title: MCP Server
appetite: large
status: shipped
shaped_at: 2026-05-22T18:05:38Z
---
## Problem

Definition docs at `.doc/definition/07-mcp-and-cli.md` and `.doc/definition/08-ai-workflow.md` describe MCP as the primary surface for AI tools. The `.mcp.json` example in `.doc/README.md` assumes `appetite mcp` exists. None of it does — `internal/mcp/` is not a package, `appetite mcp` is not a CLI subcommand, no AI tool can drive Appetite without shell-scripting around the CLI today.

Until MCP exists, the "AI-native workflow engine" thesis is hollow. The CLI proves the loop runs; MCP proves the loop runs *through the surface the product is supposed to be used from*. Cycle 1's dogfood was honest about being CLI-only — Cycle 2's cannot be.

## Appetite

3 real days. This is the parallel of 01-02's "walking skeleton" approach, applied to a second surface. The workflow package is already there; the model types are already there; typed errors are already there. What's missing is the JSON-RPC plumbing, request/response structs, and a stdio loop. 3 days is enough to wire those, get a smoke-tested end-to-end loop through MCP, and stop.

If 02-01 grows past 3 days, the contract is wrong, not the implementation. Cut tools (e.g. defer `appetite_pitch_list` / `appetite_cycle_list` to a later cycle if needed) — do not extend.

## Solution

A new `internal/mcp/` package wraps the existing `internal/workflow/` package as stdio JSON-RPC 2.0. The CLI gets one new subcommand (`appetite mcp`) that boots the server pointed at the same store. **No new workflow operations are added** — every MCP tool is a 1:1 wrapper around an existing workflow function.

```
cmd/appetite/main.go
        │
        ├─── (existing CLI dispatch)
        │
        └─── appetite mcp ──> internal/mcp/
                                  │
                                  ▼
                            internal/workflow/   (unchanged)
                                  │
                                  ▼
                            internal/store/      (unchanged)
```

### Tools shipped

Per `.doc/definition/07-mcp-and-cli.md`:

| Tool | Wraps |
|---|---|
| `appetite_status` | `workflow.Status` |
| `appetite_signal` | `workflow.Signal` |
| `appetite_shape` | `workflow.Shape` (modes: `new`, `from_signal`, `finalize`) |
| `appetite_cycle_new` | `workflow.OpenCycle` |
| `appetite_bet` | `workflow.Bet` |
| `appetite_pass` | `workflow.Pass` |
| `appetite_cut` | `workflow.Cut` |
| `appetite_hill` | `workflow.Hill` |
| `appetite_cooldown` | `workflow.Cooldown` (op: `open` w/ optional days, or `close`) |
| `appetite_pitch_list` | new read helper, calls `store.List` |
| `appetite_cycle_list` | new read helper, calls `store.List` |

### Error mapping

| Workflow error | JSON-RPC code | Mirrors CLI exit |
|---|---|---|
| `ErrInvalidTransition` | -32002 | 2 |
| `ErrDoneCriteriaUnmet` | -32002 | 2 |
| `ErrNotFound` | -32003 | 3 |
| (other) | -32603 | 1 |

The same machine-stable surface as the CLI, just translated.

### Logging discipline

Stdout is the protocol channel. All log output goes to stderr or a file (`APPETITE_MCP_LOG` env var, optional). Writing log lines to stdout corrupts the protocol — a test asserts stdout contains only protocol frames.

## Rabbit holes

- **MCP library dependency.** Resist `mark3labs/mcp-go` and similar. Stdlib `encoding/json` + a ~150-line JSON-RPC 2.0 router is enough. Mirrors 01-02's stdlib-only stance on CLI frameworks. Pulling in a framework means living with it forever.
- **Schema drift between MCP and CLI.** The request structs in `internal/workflow/` are the source of truth. MCP request types live next to them with JSON tags, not separate. If a field exists in `workflow.BetRequest` it exists in the MCP tool input — no MCP-only fields, no CLI-only fields. Divergence is the bug.
- **Streaming responses.** Defer. Every tool returns one response. If `appetite_status` ever needs to stream, it's a separate pitch.
- **Coverage floor on first contact.** 90% rule still applies. The harness pattern (write JSON-RPC frame to a `bytes.Buffer`, call `server.Handle(in, out)`, parse response, assert) covers handlers cleanly. Don't discover on day 3.
- **`appetite_pitch_list` / `appetite_cycle_list` are new code.** They're read-only and don't live in `internal/workflow/` yet — keep them tiny, in `internal/mcp/lists.go`, calling `store.List()` directly. They're not a new workflow primitive.

## No-gos

- **No HTTP server.** Stdio only. If a tool needs HTTP, it shells out to the CLI.
- **No new workflow operations.** MCP wraps. It does not extend.
- **No `--json` flag on CLI.** Defer. If MCP needs JSON output, it uses the response structs internally without exposing them on the human CLI surface yet.
- **No sync.** Cycle 3+.
- **No streaming.** Single request, single response.
- **No third-party MCP framework.** Stdlib only, same rule as 01-02.
- **No initialize/capabilities handshake beyond the JSON-RPC minimum.** If the spec mandates a `initialize` method, implement the smallest viable version; otherwise omit.

## Scope

- [ ] **MCP package skeleton + JSON-RPC loop** — `internal/mcp/server.go` with stdio JSON-RPC 2.0 framing (Content-Length headers per the MCP transport spec), method dispatch, typed error mapping in `errors.go`. Logging to stderr only; a test asserts stdout contains only protocol frames.
- [ ] **Read tools** — `appetite_status`, `appetite_pitch_list`, `appetite_cycle_list`. The status response includes cycle metadata, bets, hill positions, stuck warnings, and any sync proposals (empty slice in v0.1.x) inline. List tools accept an optional `status` filter.
- [ ] **Signal + shape tools** — `appetite_signal` (`source`, `tags`, `text`), `appetite_shape` with mode discriminator (`new` / `from_signal` / `finalize`). Finalize wraps the existing validator; an unfinalizable pitch returns `-32002` with the precise field that's missing.
- [ ] **Cycle + bet + pass tools** — `appetite_cycle_new` (id, duration), `appetite_bet` (pitch, cycle, appetite), `appetite_pass` (pitch, reason). Validation mirrors the CLI exactly — same typed errors, same error messages, same codes.
- [ ] **Cut + hill + cooldown tools** — `appetite_cut` (pitch), `appetite_hill` (card, position, progress, done), `appetite_cooldown` (op: `open` with optional days, or `close`). `appetite_hill` with `progress=100` and `done=false` returns `-32002`.
- [ ] **`appetite mcp` CLI subcommand** — `appetite mcp [--dir <path>]`, default `--dir .appetite`, reads from os.Stdin and writes to os.Stdout. Wires into the existing `internal/cli/` router. Documented in `appetite --help`; verify `.doc/definition/07-mcp-and-cli.md` matches the shipped flag surface.
- [ ] **Tests + 90% coverage floor (CI-gated)** — table-driven handler tests covering happy path + every documented error code for every tool. End-to-end test in `internal/mcp/e2e_test.go` walks the full loop via JSON-RPC (`init` → `signal` → `shape` → `cycle_new` → `bet` → `cut` → `hill` → `cooldown`). `tools/check-coverage` extends its scope to `internal/mcp/`. Sub-90% files do not merge.

---

## Reference

| Convention | Source |
|---|---|
| Workflow operations | `internal/workflow/` (shipped in 01-02) |
| Typed errors | `internal/workflow/errors.go` (shipped in 01-02) |
| Exit codes ↔ JSON-RPC codes | `.doc/definition/07-mcp-and-cli.md` |
| Coverage floor | `tools/check-coverage` (shipped in 01-02) |
| AI-workflow contract | `.doc/definition/08-ai-workflow.md` |
