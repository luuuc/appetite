---
source: operator
captured: 2026-05-23T10:48:43Z
tags: [ci, coverage, infra]
---
make ci coverage gate flakes on clean cache: parallel test binaries race on the shared coverage.out, dropping cli's positive counts; go tool cover -func then reports false-low per-function pct (e.g. targetCommandsDir 25-75% vs real 100% in the cli-only profile). Cycle 02 'passing' was a testcache artifact.
