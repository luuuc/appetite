---
slug: release-rehearsal
pitch: 01-03-dogfood-release
cycle: "01"
hill: downhill
progress: 100
hill_updated_at: 2026-05-22T16:25:27Z
---
`goreleaser release --snapshot --clean`; verify the resulting binary runs `appetite version` and one full read command (e.g. `appetite status`). Fix any config drift inline. Snapshot artifacts under `dist/` are not committed.
