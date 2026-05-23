---
description: Open or close cooldown for the active cycle.
allowed-tools: mcp__appetite__appetite_cooldown, mcp__appetite__appetite_status
---

# Cooldown

Manage the cycle's cooldown: $ARGUMENTS

Cooldown is the bounded window after a cycle ships and before the next opens. During cooldown, no new bets are accepted — polish, write lessons, draft pitches for the next cycle.

Two ops:

- `open` — flip the cycle from `shipping` → `cooldown`, optionally pass `duration` (defaults to `1d`).
- `close` — flip cooldown to `closed` and the cycle to `closed`.

## Step 1: Decide op

From `$ARGUMENTS`:

- "close" in the args → `close`
- otherwise → `open`

If ambiguous, ask.

## Step 2: Confirm — H-only

**Both cooldown ops are rule transitions.** State the op (and duration for `open`), then wait for explicit operator approval.

## Step 3: Call

Call `mcp__appetite__appetite_cooldown` with `op` (and `duration` if opening). Print the returned cooldown + cycle status.

On `-32002`, surface the violation verbatim (cycle not in shipping, no cooldown to close, pitches still building). Do not auto-resolve.

See `.doc/definition/08-ai-workflow.md` for the canonical loop semantics.
