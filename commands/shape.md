---
description: Draft, seed, or finalize an Appetite pitch with its five ingredients and scope cards.
allowed-tools: mcp__appetite__appetite_shape, mcp__appetite__appetite_status
---

# Shape

Walk the five ingredients of an Appetite pitch for: $ARGUMENTS

This is the AI-native shaping surface. A pitch carries five ingredients — **Problem, Appetite, Solution, Rabbit holes, No-gos** — plus a `## Scope` section listing the cards that complete it. The shape command supports three modes:

- **`new`** — start a pitch from scratch with a slug and title.
- **`from_signal`** — seed a pitch from an existing signal at `signals/raw/<slug>.md`.
- **`finalize`** — flip a `shaping` pitch to `shaped` (H-only rule transition).

## Step 1: Decide the mode

From `$ARGUMENTS` and the current state:

- Slug or topic provided, no existing pitch → `new`.
- Signal path mentioned → `from_signal`.
- Existing `shaping` pitch ready for review → `finalize` (but only after Step 4).

If unclear, ask the operator before doing anything.

## Step 2: Draft / refine in conversation

For `new` or `from_signal`, walk the five ingredients with the operator. Each scope card needs 3–5 `done_looks_like` entries (concrete outcomes observable from outside the code). If the operator drifts from the appetite, name it — re-shape, don't extend.

## Step 3: Write the pitch

Call `mcp__appetite__appetite_shape` with `mode: new` or `mode: from_signal` plus the slug/title/signal_path. The pitch lands as `shaping`.

## Step 4: Finalize — H-only

**Finalize is a rule transition.** Do not call `mode: finalize` without explicit operator approval ("yes, finalize" or equivalent in the conversation).

When the operator approves, call `mcp__appetite__appetite_shape` with `mode: finalize` and the slug. If the response is `-32002`, surface the missing field exactly as returned — do not paper over it.

On error, stop. See `.doc/definition/08-ai-workflow.md` for the canonical loop semantics.
