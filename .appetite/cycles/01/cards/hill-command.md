---
slug: hill-command
pitch: 01-02-workflow-engine-cli
cycle: "01"
hill: downhill
progress: 100
hill_updated_at: 2026-05-22T16:21:47Z
---
`appetite hill <card> --position <uphill|downhill> --progress <0-99>` for in-flight updates; `appetite hill <card> --progress 100 --done` for ship. Without `--done`, progress=100 exits 2. Automatic pitch→`shipped` and cycle→`shipping` transitions fire when the last card ships.
