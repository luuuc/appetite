---
slug: walk-01-02-post-hoc-tag-locally
pitch: 01-03-dogfood-release
cycle: "01"
hill: downhill
progress: 100
hill_updated_at: 2026-05-22T16:25:27Z
---
for each 01-02 card (already shipped before this pitch runs), `appetite hill <card> --position downhill --progress 100 --done` matching reality; pitch auto-flips to `shipped`. Then `git tag v0.1.0` — local only, no `git push`. Verify `appetite version` reports `v0.1.0` from the tagged binary build.
