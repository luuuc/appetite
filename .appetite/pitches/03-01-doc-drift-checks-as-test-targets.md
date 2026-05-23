---
slug: 03-01-doc-drift-checks-as-test-targets
title: Doc-Drift Checks as Test Targets
appetite: small
status: building
shaped_at: 2026-05-23T09:33:56Z
shaped_from: [2026-05-22-02-01-card-6-appetite]
---
## Problem

02-01 card 6 (appetite mcp CLI subcommand) had a checklist item: "verify .doc/definition/07-mcp-and-cli.md matches the shipped flag surface." The verification was a "check by eye" step. It was skipped. The shipped binary diverged from the doc: the doc said `appetite mcp --dir .appetite`, the code only accepted `--dir` at the top level. The drift was caught during 02-02 dogfood when `.mcp.json` failed to spawn the server — about an hour after 02-01 shipped.

This is one instance of a broader pattern. Definition docs in `.doc/definition/` describe the contract of the shipped binary; the binary's behavior is checked by tests; nothing checks the doc against the binary. A line edit on `.doc/definition/07-mcp-and-cli.md` does not need to compile, does not run a test, and only a human reading both files in the same sitting catches drift.

## Appetite

1 real day. Same shape as `tools/check-coverage` from 01-02: small Go binary, runs in CI, fails on drift. The scope is deliberately narrow — two contracts only (CLI flag surface, MCP tool list). If the parser grows past ~300 lines or the matchers turn into a templating engine, the contract is wrong, cut scope.

If 03-01 stretches because "we should also check the data-model doc", that is the second pitch, not this one.

## Solution

A new `tools/check-doc-drift/` binary that parses `.doc/definition/*.md` for two structured contracts and asserts the shipped binary matches. Wired into `make ci` next to `check-coverage`.

```
tools/check-doc-drift/main.go
        │
        ├── parse .doc/definition/07-mcp-and-cli.md
        │     ├── CLI commands + flags  (from the "## CLI" table + fenced shell blocks)
        │     └── MCP tools + params    (from the "## Tools" table)
        │
        ├── introspect the binary
        │     ├── walk `appetite <cmd> -h` for each documented command
        │     └── ask the MCP server for `tools/list`  (already shipped in 02-01)
        │
        └── diff → exit 0 if match, exit 1 with a precise "doc says X, binary says Y" report
```

### Contract 1 — CLI flag surface

`.doc/definition/07-mcp-and-cli.md` lists every `appetite` subcommand and its flags in tables and fenced shell blocks. The check:

1. Extract `appetite <cmd> [flags]` patterns from fenced ```` ```sh ```` / ```` ```bash ```` blocks.
2. For each `<cmd>`, run `appetite <cmd> -h` and parse the `-flag` lines.
3. Assert: every flag in the doc exists in the binary, and every required-looking flag in the binary appears in at least one doc example.

The card-6 drift would have been caught here: doc has `appetite mcp --dir .appetite`, binary's `mcp -h` would have shown no `--dir` flag, diff fires.

### Contract 2 — MCP tool list

`.doc/definition/07-mcp-and-cli.md` lists every MCP tool in a table (`| Tool | Wraps |`). The check:

1. Parse the tool-list table → set of expected tool names.
2. Spawn the MCP server (already-shipped `appetite mcp`), send `{"method":"tools/list"}`, parse response.
3. Assert the two sets are equal. Surplus or missing → fail with names.

### Error-message discipline

When drift is found, the failure looks like:

```
doc-drift: .doc/definition/07-mcp-and-cli.md:142 says `appetite mcp --dir <path>`
           but `appetite mcp -h` does not advertise `--dir`.
```

Same shape as `check-coverage`'s "file foo.go: 73.4% < 90% floor" — one line, both sides, file:line if available.

## Rabbit holes

- **Markdown parsing.** Use `goldmark` only if a hand-rolled scanner over fenced blocks + GFM tables gets ugly. `check-coverage` uses no parser at all — try the same first. Markdown is text; scan for the patterns that matter.
- **Runnable doc examples.** Tempting to execute every fenced ```sh``` block end-to-end. Don't. Many examples mutate state (`appetite cycle new`, `appetite bet`), and a full sandbox harness is its own pitch. This cycle: parse signatures, don't execute.
- **Schema for tool params.** The MCP `tools/list` response includes JSON-schema inputs. Checking that the doc's parameter list matches the schema is a third contract — defer. Tool-name parity is enough to catch "we forgot to ship `appetite_pitch_list`."
- **Drift between docs.** `.doc/definition/04-data-model.md` and `.doc/definition/05-storage.md` describe YAML field layouts. Tempting to validate the on-disk schema against those. Out of scope — separate pitch. This one is the CLI/MCP contract only.
- **Wiring into `make ci` vs. a new target.** Add to `make ci` directly. A separate `make check-doc-drift` target that no one runs is the same as no check.

## No-gos

- **No new docs.** Don't write `.doc/definition/09-doc-drift.md`. The tool README in `tools/check-doc-drift/` is enough; pitches and ADRs cover the rest.
- **No fixing existing drift inside this pitch.** If the check finds drift on first run, log it in a card and ship the tool first. Fixes are a separate cycle's worth of edits.
- **No third-party markdown parser unless hand-rolled scanning crosses ~300 lines.** Same stdlib-bias as 01-02 / 02-01.
- **No coverage-gating on the tool itself.** It's a one-shot CI binary, not a runtime path. `check-coverage` exempts itself the same way.
- **No HTTP, no network, no LLM.** Reads files, runs the local binary, exits.

## Scope

- [ ] **`tools/check-doc-drift/` skeleton + CI wiring** — package, `main.go`, `make ci` invocation, exit-code semantics matching `check-coverage` (0 pass, 1 drift, 2 internal error).
- [ ] **CLI flag-surface parser + checker** — extract `appetite <cmd> [flags]` from fenced shell blocks in `.doc/definition/07-mcp-and-cli.md`, diff against `appetite <cmd> -h`. Report `doc:line` ↔ binary divergence.
- [ ] **MCP tool-list checker** — parse the `| Tool | Wraps |` table, spawn `appetite mcp`, JSON-RPC `tools/list`, diff the two sets.
- [ ] **Tests + 90% coverage floor** — table-driven tests for the parser (good doc, drifted doc, malformed doc) and the diff reporter. Extend `tools/check-coverage` to include `tools/check-doc-drift/`.

---

## Reference

| Convention | Source |
|---|---|
| CI gate pattern | `tools/check-coverage/` (shipped in 01-02) |
| CLI flag surface (doc) | `.doc/definition/07-mcp-and-cli.md` |
| MCP `tools/list` | `internal/mcp/server.go` (shipped in 02-01) |
| Stdlib-only bias | 01-02 pitch + ADRs |
