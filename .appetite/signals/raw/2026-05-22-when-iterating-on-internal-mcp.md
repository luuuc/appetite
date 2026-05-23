---
source: operator
captured: 2026-05-22T18:48:43Z
tags: [coverage, tooling, friction]
---
When iterating on internal/mcp/, stale go-build cache produces stale-line-number entries in -coverprofile output; check-coverage saw functions at multiple line offsets and flagged 44% coverage when the actual state was 99%. Workaround: 'go clean -cache -testcache' before make ci. A pre-test cleanup or a deduper at location-with-tolerance in tools/check-coverage would prevent the false-positive lockup.
