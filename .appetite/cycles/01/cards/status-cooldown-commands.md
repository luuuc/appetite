---
slug: status-cooldown-commands
pitch: 01-02-workflow-engine-cli
cycle: "01"
hill: downhill
progress: 100
hill_updated_at: 2026-05-22T16:21:47Z
---
`appetite status` renders the 80-column text view (cycle line, progress bars, passed list, stuck warnings). `appetite cooldown [--days <n>]` opens cooldown for the current cycle; `appetite cooldown close` closes it. Refuses to open while pitches are still `building`.
