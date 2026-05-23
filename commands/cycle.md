---
description: Summarize the current Appetite cycle — bets, hill chart, stuck cards.
allowed-tools: mcp__appetite__appetite_status
---

# Cycle

Show the current Appetite cycle: $ARGUMENTS

Read-only status summary. No rule transitions, no operator approval needed.

## Step 1: Read

Call `mcp__appetite__appetite_status`. If `$ARGUMENTS` names a cycle id, pass it as `cycle` to look at a historical cycle.

## Step 2: Summarize

Render in this shape:

```
Cycle <id> (<duration>, day N/Total, status: <status>)

Bets:
  <progress-bar>  <pitch-slug>  <appetite>  <cards-done>/<total> cards down
  ...

Passed: <slug> (<reason>), ...

Cooldown: <status> (ends YYYY-MM-DD)

Stuck: <card-slug> — uphill <pct>%
...

Sync proposals: <none in v0.1>
```

Surface any `stuck` warnings prominently — they are the loudest signal the cycle is in trouble.

On `-32003` (no active cycle), say so and suggest `/appetite:open-cycle <id> <duration>`.

See `.doc/definition/08-ai-workflow.md` for the canonical loop semantics.
