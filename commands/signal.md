---
description: Capture a signal — raw input worth shaping into a future pitch.
allowed-tools: mcp__appetite__appetite_signal
---

# Signal

Capture an Appetite signal from: $ARGUMENTS

A signal is the smallest unit in the loop — a single sentence of observed friction, customer ask, or recurring bug. Signals land in `.appetite/signals/raw/`. Triage and prioritization happen later, during shaping.

## Step 1: Refine the note

Take `$ARGUMENTS` and shape it into a one-line title plus, optionally, a paragraph of context. Do not editorialize, do not propose solutions — a signal is the input to shaping, not the output of it.

If `$ARGUMENTS` is empty, ask the operator for a topic.

## Step 2: Pick source and tags

- `source` defaults to `operator`. Use `customer`, `beacon`, `council`, `brain`, or `manual` only when the conversation makes the origin explicit.
- `tags` are free-form — surface area, severity, or feature. One or two is plenty.

## Step 3: Write the signal

Call `mcp__appetite__appetite_signal` with `text`, `source`, and `tags`. Print the returned path so the operator can audit it.

If the call returns an error, stop and surface it — do not retry with a different shape.

See `.doc/definition/08-ai-workflow.md` for the canonical loop semantics.
