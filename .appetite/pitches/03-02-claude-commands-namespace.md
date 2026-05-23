---
slug: 03-02-claude-commands-namespace
title: Namespace Claude Slash Commands Under appetite/
appetite: small
status: shipped
shaped_at: 2026-05-23T09:36:15Z
shaped_from: [2026-05-22-appetite-init-commands-claude-shares]
---
## Problem

`appetite init --commands claude` writes eight workflow files (`signal.md`, `shape.md`, `open-cycle.md`, `bet.md`, `no-bet.md`, `break-out.md`, `cooldown.md`, `cycle.md`) directly into `.claude/commands/`. That directory is shared with whatever else lives there — including this repo's own bootstrap helpers (`shape.md`, `craft.md`, `commit.md`, `pr.md`, `council.md`) and anything the operator already had installed.

Today the installer refuses to overwrite on content drift, which is correct, but it forces a no-win choice the first time `appetite shape` and a bootstrap `shape` slash command try to share a file name. The collision is structural, not a bug in `init`: two different roles can legitimately want the same verb.

`.doc/definition/08-ai-workflow.md` currently advertises `/signal`, `/shape`, `/bet`, etc. without a prefix. The 02-02 rabbit-holes section flags this as something to fix. Until it's fixed, no third party can adopt Appetite via `appetite init --commands claude` without a manual cleanup pass — which is the opposite of what an installer is for.

## Appetite

4 real hours. The change is one directory level in the installer, one find-and-replace in eight slash-command files, one table edit in `.doc/definition/08-ai-workflow.md`, and one update to `.mcp.json` / any reference in `CLAUDE.md`. The fixture and test updates dominate; the production code is small.

If 03-02 stretches past half a day, the most likely cause is over-thinking backwards-compat for the v0.1 installs that already happened. There are exactly two such installs — this repo and Parachute's planned adoption — and both belong to the same operator. **No compat shim; rewrite and move on.**

## Solution

Adopt Claude Code's subdirectory convention: install workflow commands into `.claude/commands/appetite/`. They are invoked as `/appetite:signal`, `/appetite:shape`, `/appetite:bet`, etc.

```
before                                  after
.claude/commands/                       .claude/commands/
├── shape.md         (bootstrap)        ├── shape.md         (bootstrap, untouched)
├── shape.md         (appetite!)        ├── craft.md         (bootstrap, untouched)
├── signal.md        (appetite)         ├── ...              (bootstrap, untouched)
├── ...              (collision risk)   └── appetite/
                                            ├── signal.md
                                            ├── shape.md
                                            ├── open-cycle.md
                                            ├── bet.md
                                            ├── no-bet.md
                                            ├── break-out.md
                                            ├── cooldown.md
                                            └── cycle.md
```

### Installer change

`internal/cli/init.go` writes into `.claude/commands/appetite/` instead of `.claude/commands/`. Existing idempotence / `--force` semantics carry over unchanged. The `--commands claude` flag value stays the same — only the destination changes.

### Doc + reference updates

- `.doc/definition/08-ai-workflow.md` — every `/signal`, `/shape`, … becomes `/appetite:signal`, `/appetite:shape`, … in the surface table and prose. The Ready-to-Paste CLAUDE.md Rule Block updates with new names.
- `commands/` source files — content unchanged except cross-references between commands ("after `/bet`, run `/break-out`" → "after `/appetite:bet`, run `/appetite:break-out`").
- `CLAUDE.md` (this repo) — same find-and-replace in the Workflow section.
- `.mcp.json` — no change (server name stays `appetite`).

### Validation

A new test in `internal/cli/init_test.go` asserts:

1. `appetite init --commands claude` creates `.claude/commands/appetite/` with the eight files.
2. `.claude/commands/` is otherwise untouched (no top-level `signal.md` etc.).
3. Re-running with identical content is idempotent.
4. Re-running with diverged content refuses without `--force` (same as today).

## Rabbit holes

- **Backwards-compat for already-installed files.** Don't. v0.1 is private dogfood; both adopters are the operator. The installer detects nothing about old layouts. If a top-level `signal.md` exists, it stays — the operator deletes it by hand. Coding around it costs more than the cleanup.
- **A `commands/appetite/` source-tree mirror.** Tempting to also move `commands/` (the source files in the repo) to `commands/appetite/`. Skip. The source layout is internal; only the install destination is part of the contract. Keep `commands/*.md` flat.
- **Renaming the slash commands themselves.** `/signal` is fine inside `appetite/`. Don't rename to `/appetite-signal`-with-prefix-in-the-filename — Claude Code's subdir convention already produces the `appetite:` prefix automatically.
- **Other AI tools.** `--commands cursor`, `--commands aider`, etc. are not in this pitch. They'll each have their own namespacing decision when added.

## No-gos

- **No compat shim, no `--legacy-layout` flag.** Single layout. Document the migration as a one-line manual step.
- **No changes to the MCP tool names.** `appetite_signal` stays `appetite_signal`. The namespace problem is the Claude Code surface only.
- **No changes to the CLI surface.** `appetite signal`, `appetite shape`, etc. are untouched.
- **No docs split.** Don't introduce `.doc/definition/09-claude-namespace.md`. Update the existing table in 08 in place.
- **No `--commands cursor` in this pitch.** One installer surface at a time.

## Scope

- [ ] **Installer writes to `.claude/commands/appetite/`** — `internal/cli/init.go` change + test (`internal/cli/init_test.go`) covering create / idempotent / diverged-refuse cases.
- [ ] **Slash-command cross-references rewritten** — in every source file under `commands/`, references from one slash command to another update to the `/appetite:<name>` form. Content of each command body otherwise unchanged.
- [ ] **`.doc/definition/08-ai-workflow.md` rewrite** — table, prose, and Ready-to-Paste CLAUDE.md Rule Block all reflect the new namespace.
- [ ] **This repo's `CLAUDE.md` updated** — Workflow section uses new slash-command names. Operator runs `appetite init --commands claude --force` once to relocate the files.

---

## Reference

| Convention | Source |
|---|---|
| Installer | `internal/cli/init.go` (shipped in 02-02) |
| Slash-command source | `commands/*.md` (shipped in 02-02) |
| AI-workflow contract | `.doc/definition/08-ai-workflow.md` |
| Claude Code subdir convention | upstream Claude Code docs (subdirs produce `namespace:` prefix) |
