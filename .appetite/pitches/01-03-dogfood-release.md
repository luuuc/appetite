---
slug: 01-03-dogfood-release
title: Private Dogfood Release (v0.1.0)
appetite: ""
status: shipped
shaped_at: 2026-05-22T16:19:14Z
---
## Problem

Once 01-02 lands, Appetite can run the Shape Up loop — but nothing proves it actually does. A workflow tool that has never been used to ship itself is a hypothesis, not a product. The tag is missing, the README still describes a "pre-alpha" with "zero code," and the repo's own work (the pitches we are looking at right now) is tracked in plain Markdown rather than through Appetite's own commands.

There is no public install yet — Homebrew, announcement posts, and install.sh on a public domain are explicitly out of scope. The deliverable here is internal: prove the loop on yourself, cut a tag, update the README to the truth.

## Appetite

4 real hours. This is housekeeping plus one full dogfood pass. The bulk of the work — workflow engine, CLI, state machine — is done by 01-02. What remains is a release rehearsal, a tag, and a walkthrough of Appetite using Appetite. Anything that adds public distribution surface (Homebrew tap, hosted install.sh, blog post, HN/X) is rejected at this appetite — those belong to a later "quiet public release" cycle once the dogfood reveals what needs to change.

## Solution

Use Appetite to track Appetite's own Cycle 1, and be honest about how. 01-02 ships *before* `.appetite/` exists, so its hill positions are recorded post-hoc — a single sit-down that retrofits known reality into the workflow's notation. That is bookkeeping, not dogfooding. **The real dogfood is 01-03's own cards**: bet, cut, walked uphill/downhill, and shipped in real time as the work happens. The pitch's value is the second half — proving the loop runs while the work is in flight.

The artifact of this pitch is a committed `.appetite/` folder containing pitches, cycle.yml, and cards under `.appetite/cycles/01/`. The top-level `.doc/pitches/*.md` files remain as the source of truth for the *pitch text*; `.appetite/` tracks the *workflow state* (status, hill positions, cycle membership). The release path: `goreleaser release --snapshot --clean` for the rehearsal, then `git tag v0.1.0` locally only — no `git push`. `release.yml` stays dormant; the GitHub Releases page does not get created.

## Rabbit holes

- **README rewriting beyond v0.1 reality.** Tempting to add roadmap promises, feature matrices, comparison tables. Resist — describe only what `v0.1.0` actually does.
- **Backfilling history.** Don't try to record 01-01's bootstrap loop in `.appetite/` retroactively. Cycle 1 starts with 01-02 and 01-03.
- **Reshaping mid-dogfood.** Using Appetite will surface paper cuts. Capture them as signals (`appetite signal add`), don't reshape this pitch.
- **Goreleaser config drift.** The `.goreleaser.yml` from 01-01 may have aged. Run the snapshot early to catch breakage while there's still budget to fix it.

## No-gos

- **No `git push origin v0.1.0`.** Tag stays local. `release.yml` does not fire. The GitHub Releases page is not created.
- **No Homebrew tap.** Public distribution is a later cycle.
- **No hosted install.sh.** Local script is fine for v0.1; do not point it at a CDN or release URL yet.
- **No announcement.** No blog post, no HN, no X/Twitter, no Slack/Discord shouts.
- **No new features.** Anything that surfaces during the dogfood walkthrough becomes a signal for cycle 2.
- **No retroactive `.appetite/` history.** Cycle 1 in `.appetite/cycles/01/` covers 01-02 and 01-03 only.

## Scope

- [ ] **Init + bet + cut Cycle 1** — `appetite init` in this repo, then `appetite cycle new 01 --appetite 4d`. Bet both pitches: 01-02 large, 01-03 small. Run `appetite cut` on both; verify card files appear under `.appetite/cycles/01/cards/`. Commit the resulting skeleton with a message that explains the dogfood intent.
- [ ] **Walk 01-02 post-hoc + tag locally** — for each 01-02 card (already shipped before this pitch runs), `appetite hill <card> --position downhill --progress 100 --done` matching reality; pitch auto-flips to `shipped`. Then `git tag v0.1.0` — local only, no `git push`. Verify `appetite version` reports `v0.1.0` from the tagged binary build.
- [ ] **Release rehearsal** — `goreleaser release --snapshot --clean`; verify the resulting binary runs `appetite version` and one full read command (e.g. `appetite status`). Fix any config drift inline. Snapshot artifacts under `dist/` are not committed.
- [ ] **README and CLAUDE.md status and cycle close** — top-level `README.md` rewritten to reflect v0.1 capabilities with a "v0.1 — private dogfood. No public install yet." banner. `CLAUDE.md`'s Status section updated from "Pre-alpha. Not yet released." to "v0.1 — private dogfood; no public install." Then `appetite hill` the remaining 01-03 cards to `--done`, `appetite cooldown --days 1`, commit the resulting `.appetite/cycles/01/cooldown.yml`. Cycle closes.
