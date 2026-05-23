---
description: Propose or place a bet — flip a shaped pitch into the active cycle.
allowed-tools: mcp__appetite__appetite_bet, mcp__appetite__appetite_pitch_list, mcp__appetite__appetite_status
---

# Bet

Place an Appetite bet on: $ARGUMENTS

A bet flips a pitch from `shaped` → `bet` and appends an entry to the cycle's bets list. Bets are rule transitions — they happen only with explicit operator approval.

## Step 1: Survey shaped pitches

If `$ARGUMENTS` is empty (no slug), call `mcp__appetite__appetite_pitch_list` with `status: shaped` to surface the candidates. Present each with its title and any recommended appetite (operator's call — do not invent).

Call `mcp__appetite__appetite_status` to confirm the active cycle.

## Step 2: Propose

For a specific slug (provided or chosen from the survey), propose:

- **pitch**: `<slug>`
- **cycle**: `<active cycle id>`
- **appetite**: `micro | small | medium | large`

State that appetite is a budget, not an estimate.

## Step 3: Confirm — H-only

**Bet is a rule transition.** Wait for explicit operator approval ("yes, bet" or equivalent). Do not call MCP without it.

## Step 4: Place

Call `mcp__appetite__appetite_bet` with `pitch`, `cycle`, `appetite`. On `-32002`, surface the workflow violation verbatim (wrong status, wrong cycle). Do not retry.

See `.doc/definition/08-ai-workflow.md` for the canonical loop semantics.
