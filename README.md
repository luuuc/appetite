# Appetite

**The Shape Up methodology as a workflow engine.**

> **v0.1 — private dogfood. No public install yet.**
> Tagged locally, not published. The CLI runs the full Shape Up loop end-to-end; everything else (MCP server, PM-tool sync, public distribution) is later cycles.

Appetite turns Shape Up (by Basecamp) into a runnable workflow. Collect signals, shape pitches, bet on work, run cycles, track progress on hill charts, and ship — all from the command line. Everything is stored as Markdown and YAML files in a `.appetite/` folder.

Not a project manager. Not a ticketing system. Not a time tracker. The lightweight workflow substrate solo founders and small AI-native teams actually need.

## Build

```bash
git clone https://github.com/luuuc/appetite.git
cd appetite
make build         # → bin/appetite
```

Requires Go 1.22+. `make ci` runs build + tests + coverage gate (90% per file/function) + lint.

## The Loop

Ten commands cover one full Shape Up cycle:

| Command | What it does |
|---|---|
| `appetite init` | Create `.appetite/{signals/raw,pitches,cycles}/`. Idempotent. |
| `appetite signal add <text>` | Write a signal file. |
| `appetite shape --new\|--from\|--finalize` | Create / seed / finalize a pitch. |
| `appetite cycle new <id> --appetite <duration>` | Open a cycle in `building`. |
| `appetite bet <pitch> --cycle <id> --appetite <size>` | Place a bet (`micro`/`small`/`medium`/`large`). |
| `appetite pass <pitch> --reason <text>` | Drop a shaped pitch with a recorded reason. |
| `appetite cut <pitch>` | Read pitch `## Scope`, write one card per item. |
| `appetite hill <card> --position <p> --progress <n> [--done]` | Update hill state. `--done` required for progress=100. |
| `appetite cooldown [--days <n>] [close]` | Open or close cooldown. |
| `appetite status` | Render the active cycle: bets, hill, passed, stuck warnings. |

State transitions are validated by a state machine (`internal/model/transitions.go`). Skipping states exits 2; missing entities exit 3.

## File Layout

```
.appetite/
  signals/
    raw/                          # unprocessed signals
  pitches/
    csv-export.md                 # shaped pitch (5 ingredients + scope)
  cycles/
    2026-w15/
      cycle.yml                   # appetite, bets, status
      cards/
        export-button.md          # scope card (hill, progress)
      cooldown.yml                # written when cooldown opens
```

## Appetite Sizes

Appetite is a budget, not an estimate:

| Appetite | Budget | Typical use |
|---|---|---|
| `micro` | ~1 hour | Bug fix, copy change, config tweak |
| `small` | ~4 hours | Single feature, focused refactor |
| `medium` | ~1 day | Multi-card feature, integration work |
| `large` | ~3 days | System-level change, new subsystem |

If the work exceeds its appetite, it's a shaping failure — re-scope, don't extend.

## What Appetite Is Not

- **Not a project manager.** Runs the Shape Up loop. Does not assign tasks, track time, or manage capacity.
- **Not a backlog.** Signals that aren't shaped within a cycle are not carried forward.
- **Not a ticket system.** Scope cards are the unit of work. No sub-tasks, no story points, no velocity.

## Not Yet in v0.1

- **No MCP server.** Comes in a later cycle.
- **No PM-tool sync.** Basecamp/Linear/Notion/Trello adapters are designed (`.doc/definition/06-sync-adapters.md`) but not built.
- **No public install.** No Homebrew tap, no hosted `install.sh`, no published GitHub Release.
- **No `--json` output.** CLI is text-only for v0.1.
- **No `doctor` or sweep commands.** Validation runs in the normal write path.

## License

O'Saasy — MIT-style with SaaS-competition rights reserved. See [LICENSE](LICENSE).
