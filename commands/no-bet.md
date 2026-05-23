---
description: Pass on a shaped pitch with a recorded reason.
allowed-tools: mcp__appetite__appetite_pass, mcp__appetite__appetite_pitch_list
---

# No Bet

Pass on an Appetite pitch: $ARGUMENTS

A pass flips a pitch from `shaped` → `passed` and records the reason on the cycle. Passes are rule transitions — explicit operator approval required.

## Step 1: Pick a pitch

If `$ARGUMENTS` does not name a slug, call `mcp__appetite__appetite_pitch_list` with `status: shaped` and ask the operator which to pass.

## Step 2: Articulate the reason

Help the operator put words to *why* — "not worth it this cycle", "competing priority", "scope unclear". The reason persists in the cycle's audit trail; vague reasons compound across cycles.

## Step 3: Confirm — H-only

**Pass is a rule transition.** Wait for explicit operator approval before calling MCP.

## Step 4: Record

Call `mcp__appetite__appetite_pass` with `pitch` and `reason`. Print the returned paths. On `-32002`, surface the violation verbatim — do not infer a different status.

See `.doc/definition/08-ai-workflow.md` for the canonical loop semantics.
