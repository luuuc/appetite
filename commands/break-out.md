---
description: Cut a bet pitch into scope cards inside the active cycle.
allowed-tools: mcp__appetite__appetite_cut, mcp__appetite__appetite_status
---

# Break Out

Cut an Appetite pitch into scope cards: $ARGUMENTS

`appetite_cut` reads the pitch's `## Scope` section, writes one card per parsed line under `cycles/<cycle>/cards/`, and flips the pitch from `bet` → `building`. Cards inherit `hill: uphill, progress: 0` — the work hasn't started yet, only the framing.

## Step 1: Read the pitch's scope

Locate the pitch (`pitches/<slug>.md`) and present its current `## Scope` lines to the operator. Each line must be `- [ ] **Title** — Description`. Note any malformed lines — `appetite_cut` will reject the whole batch on a malformed entry.

## Step 2: Confirm — H-only

**Cut is a rule transition.** Wait for explicit operator approval that the scope is final and ready for cards. If the operator wants to edit scope first, stop and let them.

## Step 3: Cut

Call `mcp__appetite__appetite_cut` with `pitch: <slug>`. The response lists the created cards. Print them so the operator can verify.

On `-32002`, surface the malformed line or wrong-status detail verbatim. On `-32003`, the pitch or its bet cycle is missing — do not infer one.

See `.doc/definition/08-ai-workflow.md` for the canonical loop semantics.
