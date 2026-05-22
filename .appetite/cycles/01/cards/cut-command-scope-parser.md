---
slug: cut-command-scope-parser
pitch: 01-02-workflow-engine-cli
cycle: "01"
hill: downhill
progress: 100
hill_updated_at: 2026-05-22T16:21:47Z
---
`appetite cut <pitch>` reads the pitch's `## Scope` section via a strict line-by-line parser, writes one card file per match under `.appetite/cycles/<id>/cards/<slug>.md` with `hill: uphill / progress: 0`, then flips the pitch from `bet` to `building`. Parser errors print the offending line and exit 2; partial writes are rolled back.
