# Appetite

**The Shape Up methodology as a workflow engine.**

Appetite turns Shape Up (by Basecamp) into a runnable workflow. Collect signals, shape pitches, bet on work, run cycles, track progress on hill charts, and ship — all from the command line or any AI tool via MCP. Everything is stored as Markdown and YAML files in a `.appetite/` folder.

Not a project manager. Not a ticketing system. Not a time tracker. The lightweight workflow substrate solo founders and small AI-native teams actually need.

## Status

Pre-alpha. The workflow model is specified in `.doc/definition/`; the binary is being built.

## Install

macOS and Linux (amd64 or arm64):

```bash
curl -fsSL https://raw.githubusercontent.com/luuuc/appetite/main/install.sh | sh
```

The script detects your platform, downloads the matching binary from [GitHub Releases](https://github.com/luuuc/appetite/releases), verifies its SHA256 checksum, and installs to `/usr/local/bin/appetite` (or `~/.local/bin/appetite` if `/usr/local/bin` is not writable).

Or with Go:

```bash
go install github.com/luuuc/appetite/cmd/appetite@latest
```

## How It Works

Appetite stores everything as files in a `.appetite/` folder:

```
.appetite/
  signals/
    raw/                        # unprocessed signals
  pitches/
    csv-export.md               # shaped pitch (5 ingredients)
  cycles/
    2026-w15/
      cycle.yml                 # appetite, bets, status
      cards/
        export-button.md        # scope card
        csv-format.md           # scope card
```

### The Workflow

```
Signals ──> Shape ──> Pitch ──> Bet ──> Cycle ──> Ship ──> Cooldown
                        |                  |
                     (founder             (hill chart
                      reviews)             tracking)
```

### Appetite System

No time estimates. Appetite is a budget — how much time this is worth:

| Appetite | Budget | Typical use |
|---|---|---|
| `micro` | ~1 hour | Bug fix, copy change, config tweak |
| `small` | ~4 hours | Single feature, focused refactor |
| `medium` | ~1 day | Multi-card feature, integration work |
| `large` | ~3 days | System-level change, new subsystem |

If the work exceeds its appetite, it's a shaping failure — re-scope, don't extend.

### Hill Chart Tracking

Every card has a hill position. A card that's been uphill for half the cycle's appetite is a signal to re-scope or cut.

### The Betting Table

Not all shaped pitches get built. Passing on a pitch is not rejecting it forever — it's saying "not now." There is no backlog of passed pitches waiting for their turn.

## PM Tool Sync

Appetite is the source of truth, but most teams already live in a PM tool. Sync is bidirectional and adapter-based — Basecamp first, then Linear, Notion, Trello. **Appetite owns the workflow, the PM tool owns the view.**

## What Appetite Is Not

- **Not a project manager.** Appetite runs the Shape Up loop. It does not assign tasks, track time, or manage capacity.
- **Not a backlog.** Signals that aren't shaped within a cycle are not carried forward. If it matters, it'll come up again.
- **Not a ticket system.** Scope cards are the unit of work. There are no sub-tasks, no story points, no velocity.

## Development

```bash
make build    # build the binary
make test     # run tests
make lint     # run linters
make ci       # all of the above
```

## License

O'Saasy — MIT-style with SaaS-competition rights reserved. See [LICENSE](LICENSE).
