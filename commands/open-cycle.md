---
description: Open a new Appetite cycle in `building` status with a calendar appetite.
allowed-tools: mcp__appetite__appetite_cycle_new, mcp__appetite__appetite_status
---

# Open Cycle

Open an Appetite cycle: $ARGUMENTS

A cycle is a bounded window. Only one cycle may be open at a time. Cycle ids are the operator's choice — ISO week (`2026-w20`), quarter (`2026-q2-cycle-3`), or free-form. Appetite is a **calendar** budget (`5d`, `2w`) — distinct from per-pitch appetites.

## Step 1: Parse intent

From `$ARGUMENTS`, extract a cycle id and a duration. If either is missing, ask the operator. Do not infer.

## Step 2: Confirm

**Opening a cycle is a rule transition.** Restate the proposed id and duration to the operator. Wait for explicit approval ("yes, open it" or equivalent) before calling MCP.

If a cycle is already open, surface that to the operator and stop — closing the existing cycle is a separate decision.

## Step 3: Open

Call `mcp__appetite__appetite_cycle_new` with `id` and `duration`. Print the returned path so the operator can confirm the file on disk.

If the response is `-32002` (existing cycle still open), surface it verbatim. Do not auto-close.

See `.doc/definition/08-ai-workflow.md` for the canonical loop semantics.
