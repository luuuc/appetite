---
slug: 03-01-doc-drift-checks-as-test-targets
title: Doc-Drift Checks as Test Targets
status: shaping
shaped_from:
    - 2026-05-22-02-01-card-6-appetite
---
## Problem

02-01 card 6 (appetite mcp CLI subcommand) had a checklist item: "verify .doc/definition/07-mcp-and-cli.md matches the shipped flag surface." The verification was a "check by eye" step. It was skipped. The shipped binary diverged from the doc: the doc said `appetite mcp --dir .appetite`, the code only accepted `--dir` at the top level. The drift was caught during 02-02 dogfood when `.mcp.json` failed to spawn the server — about an hour after 02-01 shipped.

This is one instance of a broader pattern. Definition docs in `.doc/definition/` describe the contract of the shipped binary; the binary's behavior is checked by tests; nothing checks the doc against the binary. A line edit on `.doc/definition/07-mcp-and-cli.md` does not need to compile, does not run a test, and only a human reading both files in the same sitting catches drift.

## Appetite

## Solution

## Rabbit holes

## No-gos

## Scope

