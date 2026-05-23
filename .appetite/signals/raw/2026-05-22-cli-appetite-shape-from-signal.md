---
source: operator
captured: 2026-05-22T18:58:53Z
tags: [cli, shape, naming]
---
CLI 'appetite shape --from <signal>' does not accept a --slug override, so seeded pitches always inherit the signal's auto-derived slug (e.g. '2026-05-22-x'). Writing a cycle-3 pitch with the conventional '03-NN-<slug>.md' name requires either renaming the file post-creation or bypassing the CLI entirely. The MCP tool already supports an explicit slug; expose it on the CLI too.
