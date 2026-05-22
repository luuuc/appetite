---
slug: appetite-mcp-cli-subcommand
pitch: 02-01-mcp-server
cycle: "02"
hill: uphill
progress: 0
hill_updated_at: 2026-05-22T18:11:31Z
---
`appetite mcp [--dir <path>]`, default `--dir .appetite`, reads from os.Stdin and writes to os.Stdout. Wires into the existing `internal/cli/` router. Documented in `appetite --help`; verify `.doc/definition/07-mcp-and-cli.md` matches the shipped flag surface.
