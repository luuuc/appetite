---
source: operator
captured: 2026-05-22T18:49:02Z
tags: [cli, workflow, friction]
---
Card slug collisions across cycles: when two cycles ship cards with the same slug (e.g. cycle 01 and cycle 02 both have 'tests-90-coverage-floor-ci-gated'), 'appetite hill <slug>' fails until --cycle is passed. Workflow already requires this for safety, but it's friction during in-cycle dogfooding. Consider: default --cycle to the active cycle and only error when the slug exists in the active cycle's archive too.
