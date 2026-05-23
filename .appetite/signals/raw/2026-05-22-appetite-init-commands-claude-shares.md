---
source: operator
captured: 2026-05-22T18:56:06Z
tags: [slash-commands, install, namespacing]
---
appetite init --commands claude shares the .claude/commands/ namespace with bootstrap helpers (shape.md, commit.md, council.md). Without --force the install refuses on collisions, which is correct; but the rabbit-holes section of 02-02 wants the two kept separate AND the table in 08-ai-workflow.md says install to .claude/commands/. The collision surface forces operators to choose: rename one role's file or accept overwrites. Worth pitching: namespace workflow commands as 'appetite-*.md' or under '.claude/commands/appetite/'.
